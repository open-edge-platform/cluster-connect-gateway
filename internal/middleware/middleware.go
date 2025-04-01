// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/atomix/dazl"
	_ "github.com/atomix/dazl/zap"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"github.com/rancher/remotedialer"
)

var log = dazl.GetPackageLogger()

type JwtAuthenticator interface {
	ParseAndValidate(string) (jwt.Claims, error)
}

type JwtAuthorization struct {
	JwtAuthenticator JwtAuthenticator
	OpaClient        openpolicyagent.ClientWithResponsesInterface
	RbacEnabled      bool
}

func extractTunnelId(req *http.Request) (string, error) {
	segments := strings.Split(req.URL.Path, "/")
	if len(segments) >= 3 && segments[1] == "kubernetes" && segments[2] != "" {
		return segments[2], nil
	}
	return "", errors.New("invalid path format")
}

// The tunnelId string contains a UUID followed by a hyphen and a cluster name.
// Parse out the UUID and return it.
func extractProjectIdFromTunnel(tunnelId string) (string, error) {
	segments := strings.Split(tunnelId, "-")
	if len(segments) < 6 {
		return "", errors.New("invalid tunnel ID format")
	}
	projectId := strings.Join(segments[0:5], "-")
	return projectId, nil
}

func (ja *JwtAuthorization) checkOpaPolicies(req *http.Request, claims jwt.Claims) error {
	tunnelId, err := extractTunnelId(req)
	if err != nil {
		return err
	}

	projectId, err := extractProjectIdFromTunnel(tunnelId)
	if err != nil {
		return err
	}

	if claims, ok := claims.(jwt.MapClaims); ok {
		claimsMap := map[string]interface{}(claims)
		claimsMap["project_id"] = projectId

		inputJSON, err := json.Marshal(openpolicyagent.OpaInput{Input: claimsMap})
		if err != nil {
			return err
		}
		bodyReader := bytes.NewReader(inputJSON)
		resp, err := ja.OpaClient.PostV1DataPackageRuleWithBodyWithResponse(req.Context(), "rbac", "allow", &openpolicyagent.PostV1DataPackageRuleParams{}, "application/json", bodyReader)
		if err != nil {
			return err
		}
		// API reference:
		// https://github.com/open-edge-platform/orch-library/blob/main/go/pkg/openpolicyagent/openapi.yaml
		// Here we will always use "Result1" since the "allow" rule from "accessproxy" package
		// will always a boolean value.
		allowed, err := resp.JSON200.Result.AsOpaResponseResult1()
		if err != nil {
			return fmt.Errorf("unable to evaluate policy %w", err)
		}

		if allowed {
			return nil
		} else {
			return fmt.Errorf("access denied")
		}
	}
	return fmt.Errorf("Invalid JWT token")
}

func (ja *JwtAuthorization) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		token, err := request.BearerExtractor{}.ExtractToken(req)
		if err != nil {
			log.Infow("Unauthorized")
			rw.WriteHeader(http.StatusUnauthorized)
			if _, err := rw.Write([]byte(err.Error())); err != nil {
				log.Errorw("Failed to write response", dazl.Error(err))
			}
			return
		}
		claims, err := ja.JwtAuthenticator.ParseAndValidate(token)
		if err != nil {
			remotedialer.DefaultErrorWriter(rw, req, http.StatusUnauthorized, err)
			log.Infow("Unauthorized", dazl.Error(err))
			return
		}
		if ja.RbacEnabled {
			err = ja.checkOpaPolicies(req, claims)
			if err != nil {
				log.Infow("Unauthorized", dazl.Error(err))
				remotedialer.DefaultErrorWriter(rw, req, http.StatusUnauthorized, err)
				return
			}
		}

		next.ServeHTTP(rw, req)
	})
}

// SizeLimitMiddleware returns a middleware function that limits request body size
// The limit parameter specifies the maximum allowed size in bytes.
func SizeLimitMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit the size of the request body to the specified limit
			r.Body = http.MaxBytesReader(w, r.Body, limit)

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	}
}
