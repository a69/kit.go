package basic

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	httptransport "github.com/a69/kit.go/transport/http"
)

func TestWithBasicAuth(t *testing.T) {
	requiredUser := "test-user"
	requiredPassword := "test-pass"
	realm := "test realm"

	type want struct {
		result interface{}
		err    error
	}
	tests := []struct {
		name       string
		authHeader interface{}
		want       want
	}{
		{"Isn't valid with nil header", nil, want{false, AuthError{realm}}},
		{"Isn't valid with non-string header", 42, want{false, AuthError{realm}}},
		{"Isn't valid without authHeader", "", want{false, AuthError{realm}}},
		{"Isn't valid for wrong user", makeAuthString("wrong-user", requiredPassword), want{false, AuthError{realm}}},
		{"Isn't valid for wrong password", makeAuthString(requiredUser, "wrong-password"), want{false, AuthError{realm}}},
		{"Is valid for correct creds", makeAuthString(requiredUser, requiredPassword), want{true, nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.TODO(), httptransport.ContextKeyRequestAuthorization, tt.authHeader)

			result, err := AuthMiddleware[struct{}, bool](requiredUser, requiredPassword, realm)(passedValidation)(ctx, struct{}{})
			if result != tt.want.result || err != tt.want.err {
				t.Errorf("WithBasicAuth() = result: %v, err: %v, want result: %v, want error: %v", result, err, tt.want.result, tt.want.err)
			}
		})
	}
}

func makeAuthString(user string, password string) string {
	data := []byte(fmt.Sprintf("%s:%s", user, password))
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(data))
}

func passedValidation(ctx context.Context, request struct{}) (response bool, err error) {
	return true, nil
}
