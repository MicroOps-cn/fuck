package nacos

import "encoding/json"

type OnChangeFunc func(item *ConfigQueryResponse)

type ConfigParam struct {
	DataId     string `param:"dataId"`  //required
	Group      string `param:"group"`   //required
	Content    string `param:"content"` //required
	Tag        string `param:"tag"`
	AppName    string `param:"appName"`
	Desc       string `param:"desc"`
	Effect     string `param:"effect"`
	BetaIps    string `param:"betaIps"`
	CasMd5     string `param:"casMd5"`
	Type       string `param:"type"`
	Schema     string `param:"schema"`
	SrcUser    string `param:"srcUser"`
	ConfigTags string `param:"configTags"`
	OnChange   OnChangeFunc
}

type SearchConfigParam struct {
	Search   string `param:"search"`
	DataId   string `param:"dataId"`
	Group    string `param:"group"`
	Tag      string `param:"tag"`
	AppName  string `param:"appName"`
	PageNo   int    `param:"pageNo"`
	PageSize int    `param:"pageSize"`
}
type ConfigItem struct {
	Id      json.Number `param:"id"`
	DataId  string      `param:"dataId"`
	Group   string      `param:"group"`
	Content string      `param:"content"`
	Md5     string      `param:"md5"`
	Tenant  string      `param:"tenant"`
	Appname string      `param:"appname"`
}
type ConfigPage struct {
	TotalCount     int          `param:"totalCount"`
	PageNumber     int          `param:"pageNumber"`
	PagesAvailable int          `param:"pagesAvailable"`
	PageItems      []ConfigItem `param:"pageItems"`
}
