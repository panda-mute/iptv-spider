package model

import "encoding/xml"

const PrefixHeader = `<?xml version="1.0" encoding="UTF-8"?>`

/*
<?xml version="1.0" encoding="UTF-8"?>
<tv generator-info-name="" generator-info-url="">
	<channel id="28">
		<display-name lang="zh">浙江卫视</display-name>
	</channel>
	<programme start="20230223000000 +0800" stop="20230222130000 +0800" channel="20">
		<title lang="zh">Global Watch</title>
		<desc lang="zh"/>
	</programme>
</tv>
*/

// XmlTV tv tag
type XmlTV struct {
	Generator string   `xml:"generator-info-name,attr"`
	Source    string   `xml:"source-info-name,attr"`
	XMLName   xml.Name `xml:"tv"`

	Channel []*XmlTvChannel `xml:"channel"`
	Program []*Program      `xml:"programme"`
}

// XmlTvChannel : channel info
type XmlTvChannel struct {
	ID          string        `xml:"id,attr"`
	DisplayName []DisplayName `xml:"display-name"`
}

// DisplayName 频道名
type DisplayName struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

// Program 节目
type Program struct {
	Channel string `xml:"channel,attr"`
	Start   string `xml:"start,attr"`
	Stop    string `xml:"stop,attr"`

	Title []*Title `xml:"title"`
	Desc  []*Desc  `xml:"desc"`
}

// Title 节目标题
type Title struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

// Desc : 节目描述
type Desc struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}
