package model

import (
	"encoding/xml"
	"fmt"
	"github.com/golang-module/carbon"
	"testing"
)

func TestXmlGenerate(t *testing.T) {
	var now = carbon.Now()
	var xmlTv = XmlTV{
		Generator: fmt.Sprintf("%s %s", "Deny", now.ToDateTimeString()),
		Source:    "https://deny.cx",
	}

	xmlTv.Channel = append(xmlTv.Channel, &XmlTvChannel{
		ID:          "1234",
		DisplayName: []DisplayName{{Lang: "zh", Value: "浙江卫视"}},
	})

	epgBytes, err := xml.Marshal(&xmlTv)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(append([]byte(PrefixHeader), epgBytes...)))
}
