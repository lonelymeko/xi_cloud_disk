package utils

import (
    "encoding/base64"
    "strings"
)

func DecodeMaybeBase64(s string) string {
    t := strings.TrimSpace(s)
    if t == "" {
        return ""
    }
    b, err := base64.StdEncoding.DecodeString(t)
    if err == nil {
        return string(b)
    }
    return t
}
