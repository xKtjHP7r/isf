package driveradapters

import (
	"io"
	"strings"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/xeipuuv/gojsonschema"
)

const (
	_ int = iota
	i18nIDObjectsMonthlyActiveUserFileName
	i18nIDObjectsYearlyActiveUserFileName
)

// ValidateAndBindGin 校验json数据
func validateAndBindGin(c *gin.Context, schema *gojsonschema.Schema, bind interface{}) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	return validateAndBind(body, schema, bind)
}

// validateAndBind 校验json数据
func validateAndBind(body []byte, schema *gojsonschema.Schema, bind interface{}) error {
	result, err := schema.Validate(gojsonschema.NewBytesLoader(body))
	if err != nil {
		return rest.NewHTTPError(err.Error(), rest.BadRequest, nil)
	}
	if !result.Valid() {
		msgList := make([]string, 0, len(result.Errors()))
		for _, err := range result.Errors() {
			msgList = append(msgList, err.String())
		}
		return rest.NewHTTPError(strings.Join(msgList, "; "), rest.BadRequest, nil)
	}

	if err := jsoniter.Unmarshal(body, bind); err != nil {
		return err
	}

	return nil
}

// ValidateAndBindGin 校验json数据
func validateAndBindGinNewError(c *gin.Context, schema *gojsonschema.Schema, bind interface{}) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	return validateAndBindNewError(body, schema, bind)
}

// validateAndBind 校验json数据
func validateAndBindNewError(body []byte, schema *gojsonschema.Schema, bind interface{}) error {
	result, err := schema.Validate(gojsonschema.NewBytesLoader(body))
	if err != nil {
		return gerrors.NewError(gerrors.PublicBadRequest, err.Error())
	}
	if !result.Valid() {
		msgList := make([]string, 0, len(result.Errors()))
		for _, err := range result.Errors() {
			msgList = append(msgList, err.String())
		}
		return gerrors.NewError(gerrors.PublicBadRequest, strings.Join(msgList, "; "))
	}

	if err := jsoniter.Unmarshal(body, bind); err != nil {
		return err
	}

	return nil
}
