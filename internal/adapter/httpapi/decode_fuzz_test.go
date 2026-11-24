package httpapi

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// FuzzDecodeStrict checks that arbitrary request bodies never panic and that
// any body accepted by the strict decoder is a well-formed JSON object.
func FuzzDecodeStrict(f *testing.F) {
	for _, seed := range []string{
		`{"building_id":1,"number":"1","floor":0,"sq_meters":5}`,
		`{"floor":null}`,
		`{"unknown":1}`,
		`{}{}`,
		`[]`,
		``,
		`{"floor":1e400}`,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		var req apartmentRequest
		err := decodeStrict(body, &req)
		if err != nil {
			var fe *fiber.Error
			if !errors.As(err, &fe) || fe.Code != fiber.StatusBadRequest {
				t.Fatalf("decode errors must be 400 fiber errors, got %v", err)
			}
			return
		}
		if !json.Valid(body) {
			t.Fatalf("accepted invalid JSON: %q", body)
		}
		_ = req.toDomain()
	})
}
