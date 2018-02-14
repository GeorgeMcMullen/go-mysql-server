package sql

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/src-d/go-vitess/sqltypes"
	"github.com/src-d/go-vitess/vt/proto/query"
)

type Schema []*Column

func (s Schema) CheckRow(row Row) error {
	expected := len(s)
	got := len(row)
	if expected != got {
		return fmt.Errorf("expected %d values, got %d", expected, got)
	}

	for idx, f := range s {
		v := row[idx]
		if f.Check(v) {
			continue
		}

		typ := reflect.TypeOf(v).String()
		return fmt.Errorf("value at %d has unexpected type: %s",
			idx, typ)

	}

	return nil
}

// Column is the definition of a table column.
// As SQL:2016 puts it:
//   A column is a named component of a table. It has a data type, a default,
//   and a nullability characteristic.
type Column struct {
	// Name is the name of the column.
	Name string
	// Type is the data type of the column.
	Type Type
	// Default contains the default value of the column or nil if it is NULL.
	Default interface{}
	// Nullable is true if the column can contain NULL values, or false
	// otherwise.
	Nullable bool
}

func (c *Column) Check(v interface{}) bool {
	if v == nil {
		return c.Nullable
	}

	return c.Type.Check(v)
}

type Type interface {
	Name() string
	InternalType() reflect.Kind
	Check(interface{}) bool
	Convert(interface{}) (interface{}, error)
	Compare(interface{}, interface{}) int
	Native(interface{}) driver.Value
	Default() interface{}
	// Type returns the query.Type for the given Type.
	Type() query.Type
	// SQL returns the sqltypes.Value for the given value.
	SQL(interface{}) sqltypes.Value
}

var Null = nullType{}

type nullType struct{}

func (t nullType) Name() string {
	return "null"
}

func (t nullType) Type() query.Type {
	return sqltypes.Null
}

func (t nullType) SQL(interface{}) sqltypes.Value {
	return sqltypes.NULL
}

func (t nullType) InternalType() reflect.Kind {
	return reflect.Interface
}

func (t nullType) Check(v interface{}) bool {
	return v == nil
}

func (t nullType) Convert(v interface{}) (interface{}, error) {
	if v != nil {
		return nil, fmt.Errorf("value not nil: %#v", v)
	}

	return nil, nil
}

func (t nullType) Compare(a interface{}, b interface{}) int {
	//XXX: Note that while this returns 0 (equals) for ordering purposes, in
	//     SQL NULL != NULL.
	return 0
}

func (t nullType) Native(v interface{}) driver.Value {
	return driver.Value(nil)
}

func (t nullType) Default() interface{} {
	return nil
}

var Integer = integerType{}

type integerType struct{}

func (t integerType) Name() string {
	return "integer"
}

func (t integerType) Type() query.Type {
	return sqltypes.Int32
}

func (t integerType) SQL(v interface{}) sqltypes.Value {
	return sqltypes.NewInt32(MustConvert(t, v).(int32))
}

func (t integerType) InternalType() reflect.Kind {
	return reflect.Int32
}

func (t integerType) Check(v interface{}) bool {
	return checkInt32(v)
}

func (t integerType) Convert(v interface{}) (interface{}, error) {
	return convertToInt32(v)
}

func (t integerType) Compare(a interface{}, b interface{}) int {
	return compareInt32(a, b)
}

func (t integerType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(int64(v.(int32)))
}

func (t integerType) Default() interface{} {
	return int32(0)
}

var BigInteger = bigIntegerType{}

type bigIntegerType struct{}

func (t bigIntegerType) Name() string {
	return "biginteger"
}

func (t bigIntegerType) Type() query.Type {
	return sqltypes.Int64
}

func (t bigIntegerType) SQL(v interface{}) sqltypes.Value {
	return sqltypes.NewInt64(MustConvert(t, v).(int64))
}

func (t bigIntegerType) InternalType() reflect.Kind {
	return reflect.Int64
}

func (t bigIntegerType) Check(v interface{}) bool {
	return checkInt64(v)
}

func (t bigIntegerType) Convert(v interface{}) (interface{}, error) {
	return convertToInt64(v)
}

func (t bigIntegerType) Compare(a interface{}, b interface{}) int {
	return compareInt64(a, b)
}

func (t bigIntegerType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.(int64))
}

func (t bigIntegerType) Default() interface{} {
	return int64(0)
}

// TimestampWithTimezone is a timestamp with timezone.
var TimestampWithTimezone = timestampWithTimeZoneType{}

type timestampWithTimeZoneType struct{}

func (t timestampWithTimeZoneType) Name() string {
	return "timestamp with timezone"
}

func (t timestampWithTimeZoneType) Type() query.Type {
	return sqltypes.Timestamp
}

func (t timestampWithTimeZoneType) SQL(v interface{}) sqltypes.Value {
	time := MustConvert(t, v).(time.Time)
	return sqltypes.MakeTrusted(sqltypes.Timestamp,
		[]byte(time.Format("2006-01-02 15:04:05")),
	)
}

func (t timestampWithTimeZoneType) InternalType() reflect.Kind {
	return reflect.Struct
}

func (t timestampWithTimeZoneType) Check(v interface{}) bool {
	return checkTimestamp(v)
}

func (t timestampWithTimeZoneType) Convert(v interface{}) (interface{}, error) {
	return convertToTimestamp(v)
}

func (t timestampWithTimeZoneType) Compare(a interface{}, b interface{}) int {
	return compareTimestamp(a, b)
}

func (t timestampWithTimeZoneType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.(time.Time))
}

func (t timestampWithTimeZoneType) Default() interface{} {
	return time.Time{}
}

var String = stringType{}

type stringType struct{}

func (t stringType) Name() string {
	return "string"
}

func (t stringType) Type() query.Type {
	return sqltypes.Text
}

func (t stringType) SQL(v interface{}) sqltypes.Value {
	return sqltypes.MakeTrusted(sqltypes.Text, []byte(MustConvert(t, v).(string)))
}

func (t stringType) InternalType() reflect.Kind {
	return reflect.String
}

func (t stringType) Check(v interface{}) bool {
	return checkString(v)
}

func (t stringType) Convert(v interface{}) (interface{}, error) {
	return convertToString(v)
}

func (t stringType) Compare(a interface{}, b interface{}) int {
	return compareString(a, b)
}

func (t stringType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.(string))
}

func (t stringType) Default() interface{} {
	return ""
}

var Boolean Type = booleanType{}

type booleanType struct{}

func (t booleanType) Name() string {
	return "boolean"
}

func (t booleanType) Type() query.Type {
	return sqltypes.Bit
}

func (t booleanType) SQL(v interface{}) sqltypes.Value {
	b := []byte{'0'}
	if MustConvert(t, v).(bool) {
		b[0] = '1'
	}

	return sqltypes.MakeTrusted(sqltypes.Bit, b)
}

func (t booleanType) InternalType() reflect.Kind {
	return reflect.Bool
}

func (t booleanType) Check(v interface{}) bool {
	return checkBoolean(v)
}

func (t booleanType) Convert(v interface{}) (interface{}, error) {
	return convertToBool(v)
}

func (t booleanType) Compare(a interface{}, b interface{}) int {
	return compareBool(a, b)
}

func (t booleanType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.(bool))
}

func (t booleanType) Default() interface{} {
	return false
}

var Float Type = floatType{}

type floatType struct{}

func (t floatType) Name() string {
	return "float"
}

func (t floatType) InternalType() reflect.Kind {
	return reflect.Float64
}

func (t floatType) Type() query.Type {
	return sqltypes.Float64
}

func (t floatType) SQL(v interface{}) sqltypes.Value {
	return sqltypes.NewFloat64(MustConvert(t, v).(float64))
}

func (t floatType) Check(v interface{}) bool {
	return checkFloat64(v)
}

func (t floatType) Convert(v interface{}) (interface{}, error) {
	return convertToFloat64(v)
}

func (t floatType) Compare(a interface{}, b interface{}) int {
	return compareFloat64(a, b)
}

func (t floatType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.(float64))
}

func (t floatType) Default() interface{} {
	return float64(0)
}

func checkString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

func convertToString(v interface{}) (interface{}, error) {
	switch v.(type) {
	case string:
		return v.(string), nil
	case fmt.Stringer:
		return v.(fmt.Stringer).String(), nil
	default:
		return nil, ErrInvalidType
	}
}

func compareString(a interface{}, b interface{}) int {
	av := a.(string)
	bv := b.(string)
	return strings.Compare(av, bv)
}

func checkInt32(v interface{}) bool {
	_, ok := v.(int32)
	return ok
}

func convertToInt32(v interface{}) (interface{}, error) {
	switch v.(type) {
	case int:
		return int32(v.(int)), nil
	case int8:
		return int32(v.(int8)), nil
	case int16:
		return int32(v.(int16)), nil
	case int32:
		return v.(int32), nil
	case int64:
		i64 := v.(int64)
		if i64 > (1<<31)-1 || i64 < -(1<<31) {
			return nil, fmt.Errorf("value %d overflows int32", i64)
		}
		return int32(i64), nil
	case uint8:
		return int32(v.(uint8)), nil
	case uint16:
		return int32(v.(uint16)), nil
	case uint:
		u := v.(uint)
		if u > (1<<31)-1 {
			return nil, fmt.Errorf("value %d overflows int32", v)
		}
		return int32(u), nil
	case uint32:
		u := v.(uint32)
		if u > (1<<31)-1 {
			return nil, fmt.Errorf("value %d overflows int32", v)
		}
		return int32(u), nil
	case uint64:
		u := v.(uint64)
		if u > (1<<31)-1 {
			return nil, fmt.Errorf("value %d overflows int32", v)
		}
		return int32(u), nil
	case string:
		s := v.(string)
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("value %q can't be converted to int32", v)
		}
		return int32(i), nil
	default:
		return nil, ErrInvalidType
	}
}

func compareInt32(a interface{}, b interface{}) int {
	av := a.(int32)
	bv := b.(int32)
	if av < bv {
		return -1
	} else if av > bv {
		return 1
	}
	return 0
}

func checkInt64(v interface{}) bool {
	_, ok := v.(int64)
	return ok
}

func convertToInt64(v interface{}) (interface{}, error) {
	switch v.(type) {
	case int:
		return int64(v.(int)), nil
	case int8:
		return int64(v.(int8)), nil
	case int16:
		return int64(v.(int16)), nil
	case int32:
		return int64(v.(int32)), nil
	case int64:
		return v.(int64), nil
	case uint:
		return int64(v.(uint)), nil
	case uint8:
		return int64(v.(uint8)), nil
	case uint16:
		return int64(v.(uint16)), nil
	case uint32:
		return int64(v.(uint32)), nil
	case uint64:
		u := v.(uint64)
		if u >= 1<<63 {
			return nil, fmt.Errorf("value %d overflows int64", v)
		}
		return int64(u), nil
	case string:
		s := v.(string)
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("value %q can't be converted to int64", v)
		}
		return int64(i), nil
	default:
		return nil, ErrInvalidType
	}
}

func compareInt64(a interface{}, b interface{}) int {
	av := a.(int64)
	bv := b.(int64)
	if av < bv {
		return -1
	} else if av > bv {
		return 1
	}
	return 0
}

func checkBoolean(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}

func convertToBool(v interface{}) (interface{}, error) {
	switch v.(type) {
	case bool:
		return v.(bool), nil
	default:
		return nil, ErrInvalidType
	}
}

func compareBool(a interface{}, b interface{}) int {
	av := a.(bool)
	bv := b.(bool)
	if av == bv {
		return 0
	} else if av == false {
		return -1
	} else {
		return 1
	}
}

func checkFloat64(v interface{}) bool {
	_, ok := v.(float32)
	return ok
}

func convertToFloat64(v interface{}) (interface{}, error) {
	switch v.(type) {
	case float32:
		return v.(float32), nil
	default:
		return nil, ErrInvalidType
	}
}

func compareFloat64(a interface{}, b interface{}) int {
	av := a.(float32)
	bv := b.(float32)
	if av < bv {
		return -1
	} else if av > bv {
		return 1
	}
	return 0
}

func checkTimestamp(v interface{}) bool {
	_, ok := v.(time.Time)
	return ok
}

const timestampLayout = "2006-01-02 15:04:05.000000"

func convertToTimestamp(v interface{}) (interface{}, error) {
	switch value := v.(type) {
	case time.Time:
		return value, nil
	case string:
		t, err := time.Parse(timestampLayout, value)
		if err != nil {
			return nil, fmt.Errorf("value %q can't be converted to time.Time", v)
		}
		return t, nil
	default:
		if !BigInteger.Check(v) {
			return nil, ErrInvalidType
		}

		bi, err := BigInteger.Convert(v)
		if err != nil {
			return nil, ErrInvalidType
		}

		return time.Unix(bi.(int64), 0), nil
	}
}

func compareTimestamp(a interface{}, b interface{}) int {
	av := a.(time.Time)
	bv := b.(time.Time)
	if av.Before(bv) {
		return -1
	} else if av.After(bv) {
		return 1
	}
	return 0
}

var Blob = blobType{}

type blobType struct{}

func (t blobType) Name() string {
	return "blob"
}

func (t blobType) InternalType() reflect.Kind {
	return reflect.String
}

func (t blobType) Type() query.Type {
	return sqltypes.Blob
}

func (t blobType) SQL(v interface{}) sqltypes.Value {
	return sqltypes.MakeTrusted(sqltypes.Blob, MustConvert(t, v).([]byte))
}

func (t blobType) Check(v interface{}) bool {
	_, ok := v.([]byte)
	return ok
}

func (t blobType) Convert(v interface{}) (interface{}, error) {
	switch value := v.(type) {
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	case fmt.Stringer:
		return []byte(value.String()), nil
	default:
		return nil, ErrInvalidType
	}
}

func (t blobType) Compare(a interface{}, b interface{}) int {
	av := a.([]byte)
	bv := b.([]byte)
	return bytes.Compare(av, bv)
}

func (t blobType) Native(v interface{}) driver.Value {
	if v == nil {
		return driver.Value(nil)
	}

	return driver.Value(v.([]byte))
}

func (t blobType) Default() interface{} {
	return []byte{}
}

func MustConvert(t Type, v interface{}) interface{} {
	c, err := t.Convert(v)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	return c
}
