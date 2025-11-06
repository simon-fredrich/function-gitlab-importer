package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

// TODO: import YAML-file + MustStructYAML with YAML-library https://pkg.go.dev/gopkg.in/yaml.v3
var (
	observedWithWrongMessage = `{"apiVersion":"projects.gitlab.crossplane.io/v1alpha1","kind":"Project","metadata":{"annotations":{"crossplane.io/composition-resource-name":"example-project-crn","crossplane.io/external-create-failed":"2025-10-29T09:56:14Z","crossplane.io/external-create-pending":"2025-10-29T09:56:14Z","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"projects.gitlab.crossplane.io/v1alpha1\",\"kind\":\"Project\",\"metadata\":{\"annotations\":{},\"name\":\"example-project\"},\"spec\":{\"forProvider\":{\"description\":\"example project description\",\"name\":\"Example Project\",\"namespaceId\":117234999},\"providerConfigRef\":{\"name\":\"gitlab-provider\"},\"writeConnectionSecretToRef\":{\"name\":\"gitlab-project-example-project-2\",\"namespace\":\"crossplane-system\"}}}\n"},"creationTimestamp":"2025-10-28T14:52:32Z","finalizers":["finalizer.managedresource.crossplane.io"],"generation":2,"name":"example-project","resourceVersion":"277739","uid":"02e63ebf-667f-461f-aa66-438a5bd193ed"},"spec":{"deletionPolicy":"Delete","forProvider":{"description":"example project description","name":"Example Project","namespaceId":117234999,"path":"example-project"},"managementPolicies":["*"],"providerConfigRef":{"name":"gitlab-provider"},"writeConnectionSecretToRef":{"name":"example-project-secret","namespace":"crossplane-system"}},"status":{"atProvider":{},"conditions":[{"lastTransitionTime":"2025-10-29T09:56:06Z","message":"this is a wrong message","observedGeneration":2,"reason":"ReconcileError","status":"False","type":"Synced"},{"lastTransitionTime":"2025-10-28T14:53:22Z","observedGeneration":2,"reason":"Creating","status":"False","type":"Ready"}]}}`
	desiredComposedWithout   = `{"apiVersion":"projects.gitlab.crossplane.io/v1alpha1","kind":"Project","metadata":{"annotations":{"crossplane.io/composition-resource-name": "example-project-crn"},"name":"example-project"},"spec":{"forProvider":{"name":"example-project","namespaceId":117234999},"providerConfigRef":{"name":"gitlab-provider"},"writeConnectionSecretToRef":{"name":"example-project-secret","namespace":"crossplane-system"}}}`
)

// TODO: ResponseIsReturnedWithNoChange (wrong message does not change the desired resource)
// TODO: ResponseIsReturnedWithChange(right message does change the desired resource with an external-name annotation)

func TestRunFunction(t *testing.T) {

	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ResponseIsReturnedWithNoChange": {
			reason: "The Function should return the desired composed resource without any changes",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(observedWithWrongMessage),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					// TODO: add Meta, so that test runs properly
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(desiredComposedWithout),
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
