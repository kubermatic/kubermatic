/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package httproutegatewaysync

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestSanitizeListenerName(t *testing.T) {
	tests := []struct {
		name            string
		hostname        string
		want            string
		wantLen         int // expected length (0 means use len(want))
		checkHashSuffix bool
	}{
		{
			name:     "simple hostname",
			hostname: "grafana.example.com",
			want:     "grafana-example-com",
		},
		{
			name:     "wildcard hostname",
			hostname: "*.example.com",
			want:     "w-example-com",
		},
		{
			name:     "uppercase converted to lowercase",
			hostname: "Grafana.Example.COM",
			want:     "grafana-example-com",
		},
		{
			name:     "exactly 63 chars without hash",
			hostname: strings.Repeat("a", 63) + ".example.com",
			wantLen:  63,
		},
		{
			name:            "long hostname requiring truncation with hash",
			hostname:        "this-is-a-very-long-hostname-that-exceeds-the-sixty-three-character-limit.example.com",
			checkHashSuffix: true,
			wantLen:         maxListenerNameLength,
		},
		{
			name:            "long hostname with same prefix gets different hash",
			hostname:        "this-is-a-very-long-hostname-that-exceeds-the-sixty-three-character-other.example.com",
			checkHashSuffix: true,
			wantLen:         maxListenerNameLength,
		},
		{
			name:     "trailing dash in segment preserved",
			hostname: "test-.example.com",
			want:     "test--example-com",
		},
		{
			name:            "wildcard long hostname",
			hostname:        "*.this-is-a-very-long-hostname-that-exceeds-the-sixty-three-character-limit.example.com",
			checkHashSuffix: true,
			// Length may be less than maxListenerNameLength due to trailing dash trimming
		},
		{
			name:     "empty hostname",
			hostname: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeListenerName(tt.hostname)

			// verify length constraint
			if len(got) > maxListenerNameLength {
				t.Errorf("sanitizeListenerName() result too long: %d chars (max %d)", len(got), maxListenerNameLength)
			}

			// verify no trailing dash
			if strings.HasSuffix(got, "-") {
				t.Errorf("sanitizeListenerName() has trailing dash: %q", got)
			}

			// verify no leading dash (invalid DNS label)
			if len(got) > 0 && strings.HasPrefix(got, "-") {
				t.Errorf("sanitizeListenerName() has leading dash: %q", got)
			}

			// check expected length if specified
			if tt.wantLen > 0 && len(got) != tt.wantLen {
				t.Errorf("sanitizeListenerName() length = %d, want %d", len(got), tt.wantLen)
			}

			// for exact matches (non-hash cases)
			if tt.want != "" && !tt.checkHashSuffix {
				if got != tt.want {
					t.Errorf("sanitizeListenerName() = %q, want %q", got, tt.want)
				}
			}

			// verify hash suffix format for long names
			if tt.checkHashSuffix {
				parts := strings.Split(got, "-")
				if len(parts) < 2 {
					t.Errorf("expected hash suffix in %q", got)
				}
				lastPart := parts[len(parts)-1]
				if len(lastPart) != listenerNameHashLen {
					t.Errorf("hash suffix length = %d, want %d", len(lastPart), listenerNameHashLen)
				}
			}
		})
	}
}

func TestSanitizeListenerNameUniqueness(t *testing.T) {
	// two long hostnames with same 54-char prefix should produce different results
	host1 := "this-is-a-very-long-hostname-that-exceeds-the-sixty-three-character-first.example.com"
	host2 := "this-is-a-very-long-hostname-that-exceeds-the-sixty-three-character-second.example.com"

	name1 := sanitizeListenerName(host1)
	name2 := sanitizeListenerName(host2)

	if name1 == name2 {
		t.Errorf("expected different names for different hostnames, got same: %s", name1)
	}

	// verify both are within length limit
	if len(name1) > 63 || len(name2) > 63 {
		t.Errorf("names exceed 63 chars: name1=%d, name2=%d", len(name1), len(name2))
	}
}

func TestReferencesGateway(t *testing.T) {
	kindGateway := gatewayapiv1.Kind("Gateway")

	tests := []struct {
		name      string
		route     gatewayapiv1.HTTPRoute
		gatewayNS string
		want      bool
	}{
		{
			name: "references gateway in same namespace",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{Name: "kubermatic", Kind: &kindGateway},
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      true,
		},
		{
			name: "references gateway via explicit namespace",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "monitoring"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{
								Name:      "kubermatic",
								Kind:      &kindGateway,
								Namespace: (*gatewayapiv1.Namespace)(ptr.To("kubermatic")),
							},
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      true,
		},
		{
			name: "different gateway name",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{Name: "other-gateway", Kind: &kindGateway},
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      false,
		},
		{
			name: "different namespace",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "other"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{Name: "kubermatic", Kind: &kindGateway},
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      false,
		},
		{
			name: "no parent refs",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
				Spec:       gatewayapiv1.HTTPRouteSpec{},
			},
			gatewayNS: "kubermatic",
			want:      false,
		},
		{
			name: "kind not gateway",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{Name: "kubermatic", Kind: ptr.To(gatewayapiv1.Kind("NotGateway"))},
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      false,
		},
		{
			name: "nil kind defaults to gateway",
			route: gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
				Spec: gatewayapiv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
						ParentRefs: []gatewayapiv1.ParentReference{
							{Name: "kubermatic"}, // Kind is nil
						},
					},
				},
			},
			gatewayNS: "kubermatic",
			want:      true,
		},
	}

	r := &Reconciler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtw := &gatewayapiv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: tt.gatewayNS,
				},
			}
			got := r.referencesGateway(tt.route, gtw)
			if got != tt.want {
				t.Errorf("referencesGateway() = %v, want %v", got, tt.want)
			}
		})
	}
}
