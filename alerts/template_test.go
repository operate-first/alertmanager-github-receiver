// Copyright 2017 alertmanager-github-receiver Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

package alerts

import (
	"fmt"
	"html/template"
	"strings"
	"testing"
	"reflect"

	"github.com/prometheus/alertmanager/notify/webhook"
	amtmpl "github.com/prometheus/alertmanager/template"
)

func Test_formatIssueBody(t *testing.T) {
	wh := createWebhookMessage("FakeAlertName", "firing", "")
	brokenTemplate := `
{{range .NOT_REAL_FIELD}}
    * {{.Status}}
{{end}}
	`
	alertTemplate = template.Must(template.New("alert").Parse(brokenTemplate))
	got := formatIssueBody(wh)
	if got != "" {
		t.Errorf("formatIssueBody() = %q, want empty string", got)
	}
}

func TestFormatTitleSimple(t *testing.T) {
	msg := webhook.Message{
		Data: &amtmpl.Data{
			Status: "firing",
			Alerts: []amtmpl.Alert{
				{
					Annotations: amtmpl.KV{"env": "prod", "svc": "foo"},
					Labels: amtmpl.KV{"testkey1": "testvalue1a", "testkey2": "testvalue2a"},
				},
				{
					Annotations: amtmpl.KV{"env": "stage", "svc": "foo"},
					Labels: amtmpl.KV{"testkey1": "testvalue1b", "testkey2": "testvalue2b"},
				},
			},
		},
	}
	tests := []struct {
		testName	 		string
		titleTmpl      		string
		expectErrTxt 		string
		expectOutput 		string
		outputDescription	string
	}{
		{
			testName: "test static title", 
			titleTmpl: "foo", 
			expectErrTxt: "", 
			expectOutput: "foo", 
		},
		{
			testName: "test status from data in title", 
			titleTmpl: "{{.Data.Status}}", 
			expectErrTxt: "", 
			expectOutput: "firing", 
		},
		{
			testName: "test status directly in title", 
			titleTmpl: "{{.Status}}", 
			expectErrTxt: "", 
			expectOutput: "firing", 
		},
		{
			testName: "test all labels in title", 
			titleTmpl: "{{range .Alerts}}{{range .Labels}}{{.}} {{end}}{{end}}", 
			expectErrTxt: "", 
			expectOutput: "testvalue1a testvalue2a testvalue1b testvalue2b ", 
			outputDescription: "succeeds, iterates through all allerts than all labels per alert",
		},
		{
			testName: "test specific label in title", 
			titleTmpl: "{{ range .Alerts }}{{.Annotations.env}} {{ end}}", 
			expectErrTxt: "", 
			expectOutput: "prod stage ", 
			outputDescription: "succeeds, iterates through each alert and grabs the environment annotation",
		},
		{
			testName: "fail for non-existent value in title", 
			titleTmpl: "{{.Foo}}", 
			expectErrTxt: "can't evaluate field Foo in type *webhook.Message", 
			expectOutput: "", 
			outputDescription: "fails, no \"Foo\" key.",
		},
	}
	for testNum, tc := range tests {
		var labelsTmplList []string
		testNumber := fmt.Sprintf("tc=%d", testNum)
		labelsTmplList = make([]string, 4)

		labelsTmplList[0] = "{{.Labels.cluster}}"
		labelsTmplList[1] = "{{.Labels.namespace}}"
		labelsTmplList[2] = "{{.Annotations.env}}"
		labelsTmplList[3] = "{{.Annotations.svc}}"

		t.Run(testNumber, func(t *testing.T) {
			rh, err := NewReceiver(&fakeClient{}, "default", false, "", nil, tc.titleTmpl, labelsTmplList)
			if err != nil {
				t.Fatal(err)
			}
			title, err := rh.formatTitle(&msg)
			if tc.expectErrTxt == "" && err != nil {
				t.Error(err)
			}
			if tc.expectErrTxt != "" {
				if err == nil {
					t.Error()
				} else if !strings.Contains(err.Error(), tc.expectErrTxt) {
					t.Error(err.Error())
				}
			}
			if tc.expectOutput == "" && title != "" {
				t.Error(title)
			}
			if !strings.Contains(title, tc.expectOutput) {
				t.Error(title)
			}
			labels, err := rh.formatLabels(&msg)
			if err != nil {
				t.Error(labels)
			}
		})
	}
}

func TestFormatLabelsSimple(t *testing.T) {
	msg := webhook.Message{
		Data: &amtmpl.Data{
			Status: "firing",
			Alerts: []amtmpl.Alert{
				{
					Annotations: amtmpl.KV{"env": "prod", "svc": "foo"},
					Labels: amtmpl.KV{"testkey1": "testvalue1a", "testkey2": "testvalue2a"},
				},
				{
					Annotations: amtmpl.KV{"env": "stage", "svc": "foo"},
					Labels: amtmpl.KV{"testkey1": "testvalue1b", "testkey2": "testvalue2b"},
				},
			},
		},
	}
	tests := []struct {
		testName	 		string
		labelsTmpl      	string
		expectErrTxt 		string
		expectOutput 		[]string
		outputDescription	string
	}{
		{
			testName: "test static label", 
			labelsTmpl: "bar", 
			expectErrTxt: "Value does not exist.", 
			expectOutput: []string{}, 
		},
		{
			testName: "test status from data", 
			labelsTmpl: "{{.Data.Status}}", 
			expectErrTxt: "", 
			expectOutput: []string{"firing"}, 
		},
		{
			testName: "test status directly", 
			labelsTmpl: "{{.Status}}", 
			expectErrTxt: "", 
			expectOutput: []string{"firing"}, 
		},
		{
			testName: "test all labels", 
			labelsTmpl: "{{range .Alerts}}{{range .Labels}}{{.}} {{end}}{{end}}", 
			expectErrTxt: "", 
			expectOutput: []string{"testvalue1a", "testvalue2a", "testvalue1b", "testvalue2b"}, 
			outputDescription: "succeeds, iterates through all allerts than all labels per alert",
		},
		{
			testName: "test specific label", 
			labelsTmpl: "{{ range .Alerts }}{{.Annotations.env}} {{ end}}", 
			expectErrTxt: "", 
			expectOutput: []string{"prod", "stage"},
			outputDescription: "succeeds, iterates through each alert and grabs the environment annotation",
		},
		{
			testName: "fail for non-existent value", 
			labelsTmpl: "{{ .Foo }}", 
			expectErrTxt: " Value does not exist.", 
			expectOutput: []string{}, 
			outputDescription: "fails, no \"Foo\" key.",
		},
	}
	for testNum, tc := range tests {
		var labelsTmplList []string
		testNumber := fmt.Sprintf("tc=%d", testNum)
		labelsTmplList = make([]string, 1)
		labelsTmplList[0] = tc.labelsTmpl
		t.Run(testNumber, func(t *testing.T) {
			rh, err := NewReceiver(&fakeClient{}, "default", false, "", nil, tc.testName, labelsTmplList)
			if err != nil {
				t.Fatal(err)
			}
			title, err := rh.formatTitle(&msg)
			labels, err := rh.formatLabels(&msg)
			if tc.expectErrTxt == "" && err != nil {
				t.Error(err)
			}
			if tc.expectErrTxt != "" {
				if err == nil {
					t.Error()
				} else if !strings.Contains(err.Error(), tc.expectErrTxt) {
					t.Error(err.Error())
				}
			}
			if len(tc.expectOutput) == 0 && len(labels) != 0 {
				t.Error(title)
			}
			if reflect.DeepEqual(labels, tc.expectOutput) == false {
				t.Error(labels)
			}
		})
	}
}
