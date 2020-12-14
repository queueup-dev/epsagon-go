package epsagon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/epsagon/epsagon-go/protocol"
	"github.com/epsagon/epsagon-go/tracer"
)

var (
	coldStart = true
)

type genericLambdaHandler func(context.Context, json.RawMessage) (interface{}, error)

type epsagonLambdaWrapperBase struct {
	config   *Config
	tracer   tracer.Tracer
	invoked  bool
	invoking bool
}

// epsagonLambdaWrapper is a generic lambda function type
type epsagonLambdaWrapper struct {
	handler genericLambdaHandler
	epsagonLambdaWrapperBase
}

// epsagonHandlerWrapper is a implements lambda.Handler
type epsagonHandlerWrapper struct {
	awsHandler lambda.Handler
	epsagonLambdaWrapperBase
}

type preInvokeData struct {
	InvocationMetadata map[string]string
	LambdaContext      *lambdacontext.LambdaContext
	StartTime          float64
}

type invocationData struct {
	ExceptionInfo *protocol.Exception
	errorStatus   protocol.ErrorCode
	result        interface{}
	err           error
	thrownError   interface{}
}

func getAWSAccount(lc *lambdacontext.LambdaContext) string {
	arnParts := strings.Split(lc.InvokedFunctionArn, ":")
	if len(arnParts) >= 4 {
		return arnParts[4]
	}
	return ""
}

func (wrapper *epsagonLambdaWrapperBase) preInvokeOps(
	ctx context.Context, payload json.RawMessage) (info *preInvokeData) {
	startTime := tracer.GetTimestamp()
	metadata := map[string]string{}
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		lc = &lambdacontext.LambdaContext{}
	}
	defer func() {
		if r := recover(); r != nil {
			wrapper.tracer.AddExceptionTypeAndMessage("LambdaWrapper",
				fmt.Sprintf("preInvokeOps:%+v", r))
			info = &preInvokeData{
				LambdaContext:      lc,
				StartTime:          startTime,
				InvocationMetadata: metadata,
			}
		}
	}()

	metadata = map[string]string{
		"log_stream_name":  lambdacontext.LogStreamName,
		"log_group_name":   lambdacontext.LogGroupName,
		"function_version": lambdacontext.FunctionVersion,
		"memory":           strconv.Itoa(lambdacontext.MemoryLimitInMB),
		"cold_start":       strconv.FormatBool(coldStart),
		"aws_account":      getAWSAccount(lc),
		"region":           os.Getenv("AWS_REGION"),
	}
	coldStart = false

	addLambdaTrigger(payload, wrapper.config.MetadataOnly, triggerFactories, wrapper.tracer)

	return &preInvokeData{
		InvocationMetadata: metadata,
		LambdaContext:      lc,
		StartTime:          startTime,
	}
}

func (wrapper *epsagonLambdaWrapperBase) postInvokeOps(
	preInvokeInfo *preInvokeData,
	invokeInfo *invocationData) {
	defer func() {
		if r := recover(); r != nil {
			wrapper.tracer.AddExceptionTypeAndMessage("LambdaWrapper", fmt.Sprintf("postInvokeOps:%+v", r))
		}
	}()

	endTime := tracer.GetTimestamp()
	duration := endTime - preInvokeInfo.StartTime

	lambdaEvent := &protocol.Event{
		Id:        preInvokeInfo.LambdaContext.AwsRequestID,
		StartTime: preInvokeInfo.StartTime,
		Resource: &protocol.Resource{
			Name:      lambdacontext.FunctionName,
			Type:      "lambda",
			Operation: "invoke",
			Metadata:  preInvokeInfo.InvocationMetadata,
		},
		Origin:    "runner",
		Duration:  duration,
		ErrorCode: invokeInfo.errorStatus,
		Exception: invokeInfo.ExceptionInfo,
	}

	if !wrapper.config.MetadataOnly {
		if result, ok := invokeInfo.result.([]byte); ok {
			lambdaEvent.Resource.Metadata["return_value"] = string(result)
		} else {
			result, err := json.Marshal(invokeInfo.result)
			if err == nil {
				lambdaEvent.Resource.Metadata["return_value"] = string(result)
			} else {
				lambdaEvent.Resource.Metadata["return_value"] = fmt.Sprintf("%+v", invokeInfo.result)
			}
		}
	}

	wrapper.tracer.AddEvent(lambdaEvent)
}

// Invoke calls the wrapper, and creates a tracer for that duration.
func (wrapper *epsagonLambdaWrapper) Invoke(ctx context.Context, payload json.RawMessage) (result interface{}, err error) {
	invokeInfo := &invocationData{}
	wrapper.invoked = false
	wrapper.invoking = false
	defer func() {
		if !wrapper.invoking {
			recover()
		}
		if !wrapper.invoked {
			result, err = wrapper.handler(ctx, payload)
		}
		if invokeInfo.thrownError != nil {
			panic(userError{
				exception: invokeInfo.thrownError,
				stack:     invokeInfo.ExceptionInfo.Traceback,
			})
		}
	}()

	preInvokeInfo := wrapper.preInvokeOps(ctx, payload)
	wrapper.InvokeClientLambda(ctx, payload, invokeInfo)
	wrapper.postInvokeOps(preInvokeInfo, invokeInfo)

	return invokeInfo.result, invokeInfo.err
}

func (wrapper *epsagonHandlerWrapper) Invoke(ctx context.Context, payload []byte) (result []byte, err error) {
	invokeInfo := &invocationData{}
	wrapper.invoked = false
	wrapper.invoking = false
	defer func() {
		if !wrapper.invoking {
			recover()
		}
		if !wrapper.invoked {
			result, err = wrapper.awsHandler.Invoke(ctx, payload)
		}
		if invokeInfo.thrownError != nil {
			panic(userError{
				exception: invokeInfo.thrownError,
				stack:     invokeInfo.ExceptionInfo.Traceback,
			})
		}
	}()

	preInvokeInfo := wrapper.preInvokeOps(ctx, payload)
	wrapper.InvokeClientLambda(ctx, payload, invokeInfo)
	wrapper.postInvokeOps(preInvokeInfo, invokeInfo)

	return invokeInfo.result.([]byte), invokeInfo.err
}

func (wrapper *epsagonLambdaWrapper) InvokeClientLambda(
	ctx context.Context, payload json.RawMessage, invokeInfo *invocationData) {
	defer func() {
		invokeInfo.thrownError = recover()
		if invokeInfo.thrownError != nil {
			invokeInfo.ExceptionInfo = &protocol.Exception{
				Type:      "Runtime Error",
				Message:   fmt.Sprintf("%v", invokeInfo.thrownError),
				Traceback: string(debug.Stack()),
				Time:      tracer.GetTimestamp(),
			}
			invokeInfo.errorStatus = protocol.ErrorCode_EXCEPTION
		}
	}()

	invokeInfo.errorStatus = protocol.ErrorCode_OK
	// calling the actual function:
	wrapper.invoked = true
	wrapper.invoking = true
	result, err := wrapper.handler(ctx, payload)
	wrapper.invoking = false
	if err != nil {
		invokeInfo.errorStatus = protocol.ErrorCode_ERROR
		invokeInfo.ExceptionInfo = &protocol.Exception{
			Type:      "Error Result",
			Message:   err.Error(),
			Traceback: "",
			Time:      tracer.GetTimestamp(),
		}
	}
	invokeInfo.result = result
	invokeInfo.err = err
}

func (wrapper *epsagonHandlerWrapper) InvokeClientLambda(
	ctx context.Context, payload json.RawMessage, invokeInfo *invocationData) {
	defer func() {
		invokeInfo.thrownError = recover()
		if invokeInfo.thrownError != nil {
			invokeInfo.ExceptionInfo = &protocol.Exception{
				Type:      "Runtime Error",
				Message:   fmt.Sprintf("%v", invokeInfo.thrownError),
				Traceback: string(debug.Stack()),
				Time:      tracer.GetTimestamp(),
			}
			invokeInfo.errorStatus = protocol.ErrorCode_EXCEPTION
		}
	}()

	invokeInfo.errorStatus = protocol.ErrorCode_OK
	// calling the actual function:
	wrapper.invoked = true
	wrapper.invoking = true
	result, err := wrapper.awsHandler.Invoke(ctx, payload)
	wrapper.invoking = false
	if err != nil {
		invokeInfo.errorStatus = protocol.ErrorCode_ERROR
		invokeInfo.ExceptionInfo = &protocol.Exception{
			Type:      "Error Result",
			Message:   err.Error(),
			Traceback: "",
			Time:      tracer.GetTimestamp(),
		}
	}
	invokeInfo.result = result
	invokeInfo.err = err
}

// WrapLambdaHandler wraps a generic wrapper for lambda function with epsagon tracing
func WrapLambdaHandler(config *Config, handler interface{}) interface{} {
	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		wrapperTracer := tracer.CreateGlobalTracer(&config.Config)
		wrapperTracer.Start()
		defer wrapperTracer.Stop()
		wrapper := &epsagonLambdaWrapper{
			handler:                  makeGenericHandler(handler),
			epsagonLambdaWrapperBase: epsagonLambdaWrapperBase{tracer: wrapperTracer, config: config},
		}

		return wrapper.Invoke(ctx, payload)
	}
}

// WrapLambdaHandler wraps a lambda.Handler with epsagon tracing
func WrapHandler(config *Config, handler lambda.Handler) lambda.Handler {
	wrapperTracer := tracer.CreateGlobalTracer(&config.Config)
	wrapperTracer.Start()
	defer wrapperTracer.Stop()
	wrapper := &epsagonHandlerWrapper{
		awsHandler:               handler,
		epsagonLambdaWrapperBase: epsagonLambdaWrapperBase{tracer: wrapperTracer, config: config},
	}

	return wrapper
}
