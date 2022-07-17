/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"fmt"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KubeAwareEncoder is based on sigs.k8s.io/controller-runtime/pkg/log/zap.KubeAwareEncoder,
// but uses the stringified format for object references as described in
// https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/1602-structured-logging#object-reference-format-in-logs
type KubeAwareEncoder struct {
	// Encoder is the zapcore.Encoder that this encoder delegates to
	zapcore.Encoder
}

// Clone implements zapcore.Encoder.
func (k *KubeAwareEncoder) Clone() zapcore.Encoder {
	return &KubeAwareEncoder{
		Encoder: k.Encoder.Clone(),
	}
}

// EncodeEntry implements zapcore.Encoder.
func (k *KubeAwareEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	for i, field := range fields {
		if field.Type == zapcore.StringerType || field.Type == zapcore.ReflectType {
			switch val := field.Interface.(type) {
			case runtime.Object:
				encoded, err := k.encodeObject(val)
				if err != nil {
					return nil, fmt.Errorf("failed to encode %v: %w", field.Interface, err)
				}

				fields[i] = zapcore.Field{
					Type:   zapcore.StringType,
					Key:    field.Key,
					String: encoded,
				}

			case types.NamespacedName:
				fields[i] = zapcore.Field{
					Type:   zapcore.StringType,
					Key:    field.Key,
					String: k.encodeNamespacedName(val),
				}

			case reconcile.Request:
				fields[i] = zapcore.Field{
					Type:   zapcore.StringType,
					Key:    field.Key,
					String: k.encodeNamespacedName(val.NamespacedName),
				}
			}
		}
	}

	return k.Encoder.EncodeEntry(entry, fields)
}

func (k *KubeAwareEncoder) encodeNamespacedName(name types.NamespacedName) string {
	if name.Namespace == "" {
		return name.Name
	}

	return fmt.Sprintf("%s/%s", name.Namespace, name.Name)
}

func (k *KubeAwareEncoder) encodeObject(obj runtime.Object) (string, error) {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return "", fmt.Errorf("got runtime.Object without object metadata: %v", obj)
	}

	return k.encodeNamespacedName(types.NamespacedName{
		Namespace: objMeta.GetNamespace(),
		Name:      objMeta.GetName(),
	}), nil
}
