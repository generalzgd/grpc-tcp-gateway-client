/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: fun.go
 * @time: 2019/9/9 17:07
 */
package sub

import (
	gwproto `github.com/generalzgd/grpc-tcp-gateway-proto/goproto`
	`github.com/golang/protobuf/proto`
)

func FormatObj(cmdid uint16, body []byte) string {
	if obj := gwproto.GetMsgObjById(cmdid); obj != nil {
		if err := proto.Unmarshal(body, obj); err == nil {
			return obj.String()
		}
	}
	return ""
}
