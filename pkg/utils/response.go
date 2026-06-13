package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "ok", data)
}

func Fail(w http.ResponseWriter, code int, msg string) {
	JSON(w, http.StatusOK, code, msg, nil)
}

func JSON(w http.ResponseWriter, httpCode, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpCode)
	_ = json.NewEncoder(w).Encode(Resp{Code: code, Msg: msg, Data: data})
}

func RandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func MD5File(data []byte) string {
	s := md5.Sum(data)
	return hex.EncodeToString(s[:])
}

func RenderTemplate(tpl string, vars map[string]string) string {
	result := tpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", k), v)
	}
	return result
}
