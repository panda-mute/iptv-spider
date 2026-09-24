package auth

import (
	"iptv-spider/model"
	"reflect"
	"strings"
)

func GetChannelFormString(c string) model.Channel {
	data := map[string]interface{}{}
	// 分割字符串
	sp := strings.Split(c, ",")
	for _, s := range sp {
		c := strings.SplitN(s, "=", 2)
		//key := strings.ToLower(c[0])
		key := c[0]
		data[key] = strings.Trim(c[1], `"`)
	}
	ch := model.Channel{}
	channelBindFormMap(data, &ch)
	return ch
}

func channelBindFormMap(data map[string]interface{}, dst interface{}) {
	rType := reflect.TypeOf(dst)
	rVal := reflect.ValueOf(dst)
	if rType.Kind() != reflect.Pointer || rVal.IsNil() {
		panic("inStructPtr must be ptr to struct")
	}
	// 传入的dst是指针，需要.Elem()取得指针指向的value
	rType = rType.Elem()
	rVal = rVal.Elem()

	for i := 0; i < rType.NumField(); i++ {
		t := rType.Field(i)
		f := rVal.Field(i)
		//得到tag中的字段名
		key := t.Tag.Get("key")
		if key == "-" {
			continue
		}
		//key := strings.ToLower(t.Name)
		if v, ok := data[key]; ok {
			// 检查是否需要类型转换
			dataType := reflect.TypeOf(v)
			structType := f.Type()
			if structType == dataType {
				f.Set(reflect.ValueOf(v))
			} else {
				if dataType.ConvertibleTo(structType) {
					// 转换类型
					f.Set(reflect.ValueOf(v).Convert(structType))
				}
			}
		}
	}
}
