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
	"bytes"
	"fmt"
	"github.com/prometheus/alertmanager/notify/webhook"
	"html/template"
	"log"
	"strings"
)

const (
	// alertMD reports all alert labels and annotations in a markdown format
	// that renders correctly in github issues.
	//
	// Example:
	//
	// Alertmanager URL: http://localhost:9093
	//
	//  * firing
	//
	//	Labels:
	//
	//	 - alertname = DiskRunningFull
	//	 - dev = sda1
	//	 - instance = example1
	//
	//	Annotations:
	//
	//	 - test = value
	//
	//  * firing
	//
	//	Labels:
	//
	//	 - alertname = DiskRunningFull
	//	 - dev = sda2
	//   - instance = example2
	alertMD = `
Alertmanager URL: {{.Data.ExternalURL}}
{{range .Data.Alerts}}
  * {{.Status}} {{.GeneratorURL}}
  {{if .Labels}}
    Labels:
  {{- end}}
  {{range $key, $value := .Labels}}
    - {{$key}} = {{$value -}}
  {{end}}
  {{if .Annotations}}
    Annotations:
  {{- end}}
  {{range $key, $value := .Annotations}}
    - {{$key}} = {{$value -}}
  {{end}}
{{end}}

TODO: add graph url from annotations.
`

	// DefaultTitleTmpl will be used to format the title string if it's not
	// overridden.
	DefaultTitleTmpl = `{{ .Data.GroupLabels.alertname }}`
)

var (
	alertTemplate = template.Must(template.New("alert").Parse(alertMD))
)

func id(msg *webhook.Message) string {
	return fmt.Sprintf("0x%x", msg.GroupKey)
}

// formatTitle constructs an issue title from a webhook message.
func (rh *ReceiverHandler) formatTitle(msg *webhook.Message) (string, error) {
	var title bytes.Buffer
	if err := rh.titleTmpl.Execute(&title, msg); err != nil {
		return "", err
	}
	return title.String(), nil
}

// formatIssueBody constructs an issue body from a webhook message.
func formatIssueBody(msg *webhook.Message) string {
	var buf bytes.Buffer
	err := alertTemplate.Execute(&buf, msg)
	if err != nil {
		log.Printf("Error executing template: %s", err)
		return ""
	}
	s := buf.String()
	return fmt.Sprintf("<!-- ID: %s -->\n%s", id(msg), s)
}

// formatIssueBody constructs a github labels from a webhook message.
func (rh *ReceiverHandler) formatLabels(msg *webhook.Message) ([]string, error) {
	var labelBuff bytes.Buffer
	var ghLabels []string
	var contextCheck string
	var labelTemplate bool
	if len(rh.labelsTmpl) == 1 && rh.LabelsTmplList[0][0:2] == "{{" { // if length is 1 for labels and checking for template characters
		labelTemplate = true
	} else {
		labelTemplate = false
	}
	for index, label := range rh.labelsTmpl { 
		stringLabel := rh.LabelsTmplList[index]
		propertyDepth := len(strings.Split(rh.LabelsTmplList[index], "."))-1
		if propertyDepth > 1 {
			firstPropertyIndex := strings.Index(stringLabel, ".")
			secondPropertyIndex := strings.Index(stringLabel[firstPropertyIndex+1:], ".") + firstPropertyIndex + 1
			firstCloseIndex := strings.Index(stringLabel, "}")
			if secondPropertyIndex < firstCloseIndex {
				contextCheck = stringLabel[firstPropertyIndex:secondPropertyIndex]
			} else {
				contextCheck = stringLabel[firstPropertyIndex:firstCloseIndex]
			}
		} else if propertyDepth == 1 { 
			firstPropertyIndex := strings.Index(stringLabel, ".")
			contextCheck = stringLabel[firstPropertyIndex:strings.Index(stringLabel, "}")]
		} else {
			contextCheck = rh.LabelsTmplList[index]
		}
		contextCheck = strings.TrimSpace(contextCheck)
		
		if contextCheck == ".Data" {
			if err := label.Execute(&labelBuff, &msg); err != nil {
				return []string{}, err
			}
		} else if contextCheck == ".Alerts" || contextCheck == ".Status" {
			if err := label.Execute(&labelBuff, &msg.Data); err != nil {
				return []string{}, err
			}
		} else if contextCheck == ".Labels" || contextCheck == ".Annotations"  {
			if err := label.Execute(&labelBuff, &msg.Data.Alerts[0]); err != nil {
				return []string{}, err
			} 	
		} else {
			return []string{}, fmt.Errorf("no valid context to use for label.execute. Value does not exist.")
		}
		ghlabel := labelBuff.String()
		ghlabel = strings.TrimSpace(ghlabel)
		if ghlabel != "" {
			ghLabels = append(ghLabels, ghlabel)
		}
		labelBuff.Reset()
	}
	if labelTemplate == true {
		ghLabels = strings.Split(ghLabels[0], " ")
	}
	return ghLabels, nil
}
