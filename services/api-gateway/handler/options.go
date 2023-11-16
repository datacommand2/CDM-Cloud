package handler

import (
	"github.com/datacommand2/cdm-cloud/services/api-gateway/handler/wrapper"
)

// Options handler wrapping을 위한 옵션
type Options struct {
	wrappers []wrapper.Wrapper
}

// Option gateway Options 값 설정
type Option func(o *Options)

// WithWrapper apigateway wrapper handler 추가
func WithWrapper(w wrapper.Wrapper) Option {
	return func(o *Options) {
		o.wrappers = append(o.wrappers, w)
	}
}
