package v1beta1

import (
	"github.com/kserve/kserve/pkg/apis/serving/v1beta1"
)

// 只重用KServe的类型定义，不注册到scheme
type InferenceService = v1beta1.InferenceService
type InferenceServiceSpec = v1beta1.InferenceServiceSpec
type InferenceServiceStatus = v1beta1.InferenceServiceStatus
