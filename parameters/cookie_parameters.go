// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package parameters

import (
    "github.com/pb33f/libopenapi-validator/errors"
    "github.com/pb33f/libopenapi-validator/helpers"
    "github.com/pb33f/libopenapi-validator/paths"
    "github.com/pb33f/libopenapi-validator/schemas"
    "github.com/pb33f/libopenapi/datamodel/high/base"
    "net/http"
    "strconv"
    "strings"
)

// ValidateCookieParams validates the cookie parameters contained within *http.Request.
// It returns a boolean stating true if validation passed (false for failed),
// and a slice of errors if validation failed.
func (v *paramValidator) ValidateCookieParams(request *http.Request) (bool, []*errors.ValidationError) {

    // find path
    pathItem, errs, _ := paths.FindPath(request, v.document)
    if pathItem == nil || errs != nil {
        v.errors = errs
        return false, errs
    }

    // extract params for the operation
    var params = helpers.ExtractParamsForOperation(request, pathItem)
    var validationErrors []*errors.ValidationError
    for _, p := range params {
        if p.In == helpers.Cookie {
            for _, cookie := range request.Cookies() {
                if cookie.Name == p.Name { // cookies are case-sensitive, an exact match is required

                    var sch *base.Schema
                    if p.Schema != nil {
                        sch = p.Schema.Schema()
                    }
                    pType := sch.Type

                    for _, ty := range pType {
                        switch ty {
                        case helpers.Integer, helpers.Number:
                            if _, err := strconv.ParseFloat(cookie.Value, 64); err != nil {
                                validationErrors = append(validationErrors,
                                    errors.InvalidCookieParamNumber(p, strings.ToLower(cookie.Value), sch))
                            }
                        case helpers.Boolean:
                            if _, err := strconv.ParseBool(cookie.Value); err != nil {
                                validationErrors = append(validationErrors,
                                    errors.IncorrectCookieParamBool(p, strings.ToLower(cookie.Value), sch))
                            }
                        case helpers.Object:
                            if !p.IsExploded() {
                                var encodedObj interface{}
                                encodedObj = helpers.ConstructMapFromCSV(cookie.Value)

                                // if a schema was extracted
                                if sch != nil {
                                    validationErrors = append(validationErrors,
                                        schemas.ValidateParameterSchema(sch, encodedObj, "",
                                            "Cookie parameter",
                                            "The cookie parameter",
                                            p.Name,
                                            helpers.ParameterValidation,
                                            helpers.ParameterValidationQuery)...)
                                }

                            }
                        case helpers.Array:

                            if !p.IsExploded() {
                                // well we're already in an array, so we need to check the items schema
                                // to ensure this array items matches the type
                                // only check if items is a schema, not a boolean
                                if sch.Items.IsA() {
                                    validationErrors = append(validationErrors,
                                        ValidateCookieArray(sch, p, cookie.Value)...)
                                }
                            }

                        case helpers.String:

                            // check if the schema has an enum, and if so, match the value against one of
                            // the defined enum values.
                            if sch.Enum != nil {
                                matchFound := false
                                for _, enumVal := range sch.Enum {
                                    if strings.TrimSpace(cookie.Value) == enumVal {
                                        matchFound = true
                                        break
                                    }
                                }
                                if !matchFound {
                                    validationErrors = append(validationErrors,
                                        errors.IncorrectCookieParamEnum(p, strings.ToLower(cookie.Value), sch))
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    if len(validationErrors) > 0 {
        return false, validationErrors
    }
    return true, nil
}
