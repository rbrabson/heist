package shop

/*
	This is almost certainly not going to be used. But it is an interesting example of using
	a custom type for JSON and BSON marshalling and unmarshalling.
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// Duration allows custom marshalling and unmarshalling of the duration
type Duration time.Duration

// MarshalJSON returns the duration as a string
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshallJSON parses the input `float64` or `string` into a Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// MarshalBSONValue marshalls a Duration into a bson string
func (d Duration) MarshalBSONValue() (bsontype.Type, []byte, error) {
	durationStr := time.Duration(d).String()
	return bson.TypeString, bsoncore.AppendString(nil, durationStr), nil
}

// UnmarshalBSONValue unmarshalles a bson value into a Duration
func (d *Duration) UnmarshalBSONValue(btype bsontype.Type, data []byte) error {
	switch btype {
	case bson.TypeDouble:
		value, _, ok := bsoncore.ReadDouble(data)
		if !ok {
			return fmt.Errorf("invalid bson double value")
		}
		*d = Duration(time.Duration(value))
		return nil
	case bson.TypeString:
		value, _, ok := bsoncore.ReadString(data)
		if !ok {
			return fmt.Errorf("invalid bson string value")
		}
		timeDuration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		fmt.Println(timeDuration)
		*d = Duration(timeDuration)
		return nil
	default:
		return errors.New("invalid duration")
	}
}
