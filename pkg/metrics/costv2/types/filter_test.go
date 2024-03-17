package costv2

import (
	"reflect"
	"testing"
)

func TestParseFilter(t *testing.T) {
	// 定义测试用例
	tests := []struct {
		name      string
		filterStr string
		want      *Filter
		wantErr   bool
	}{
		{
			name:      "Valid filter string",
			filterStr: `namespace:"default,kube-system"+controllerName:"deployment,daemonset"+controllerKind:"deployment,daemonset"+pod:"qwqwqwqw,qdqd23124e!@!$$%#$%"+label[app+1q]:"nginx-/!@#+_)(webserver"`,
			want: &Filter{
				Namespace:      []string{"default", "kube-system"},
				ControllerName: []string{"deployment", "daemonset"},
				ControllerKind: []string{"deployment", "daemonset"},
				Pod:            []string{"qwqwqwqw", "qdqd23124e!@!$$%#$%"},
				Label:          map[string][]string{"app+1q": []string{"nginx-/!@#+_)(webserver"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilter(tt.filterStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFilter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
