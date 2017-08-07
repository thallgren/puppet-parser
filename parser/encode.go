// +build go1.7

package parser

import (
  "encoding/json"
  "io"
)

func Encode(expr Expression, result io.Writer) {
  enc := json.NewEncoder(result)
  enc.SetEscapeHTML(false)
  enc.Encode(expr.ToPN().ToData())
}
