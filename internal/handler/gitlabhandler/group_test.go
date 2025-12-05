package gitlabhandler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/simon-fredrich/function-gitlab-importer/internal/testutils"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/crossplane/function-sdk-go/resource"
)

func TestGetPath(t *testing.T) {
	type args struct {
		des *resource.DesiredComposed
	}

	type want struct {
		pathString string
		err        error
	}

	var filename string
	filename = "group-with-path.json"
	desWithPath, err := testutils.LoadDesiredComposedFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load test data %s: %v", filename, err)
	}

	filename = "group-without-path.json"
	desWithoutPath, err := testutils.LoadDesiredComposedFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load test data %s: %v", filename, err)
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"TestDesiredWithPath": {
			reason: "Check if function works with a correct desired.",
			args: args{
				des: desWithPath,
			},
			want: want{
				pathString: "group-to-import",
				err:        nil,
			},
		},
		"TestDesiredWithoutPath": {
			reason: "Check if function failes with an incorrect desired.",
			args: args{
				des: desWithoutPath,
			},
			want: want{
				pathString: "",
				err:        cmpopts.AnyError,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g := &GroupHandler{}
			pathString, err := g.GetPath(tc.args.des)

			if diff := cmp.Diff(tc.want.pathString, pathString, protocmp.Transform()); diff != "" {
				t.Errorf("%s\ng.GetPath(...): -want pathString, +got pathString:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\ng.GetPath(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
