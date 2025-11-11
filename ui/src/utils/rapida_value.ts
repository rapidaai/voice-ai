import { Map } from 'google-protobuf';
import { Any } from 'google-protobuf/google/protobuf/any_pb';
import * as google_protobuf_any_pb from 'google-protobuf/google/protobuf/any_pb';
import { ListValue, Value } from 'google-protobuf/google/protobuf/struct_pb';
import {
  StringValue,
  Int32Value,
  DoubleValue,
  BoolValue,
  BytesValue,
} from 'google-protobuf/google/protobuf/wrappers_pb';

// Helper function to pack data into `Any`
function pack(serialized: Uint8Array, typeUrlPrefix: string): Any {
  const anyValue = new Any();
  anyValue.pack(serialized, typeUrlPrefix);
  return anyValue;
}

//  string to `Any`
export function StringToAny(value: string): Any {
  const stringValue = new StringValue();
  stringValue.setValue(value);
  const serialized = stringValue.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.StringValue');
}

//  `Any` to string
export function AnyToString(anyValue: Any): string {
  const stringValue = StringValue.deserializeBinary(anyValue.getValue_asU8());
  return stringValue.getValue();
}

//  float to `Any`
export function FloatToAny(value: number): Any {
  const doubleValue = new DoubleValue();
  doubleValue.setValue(value);
  const serialized = doubleValue.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.DoubleValue');
}

//  `Any` to float
export function AnyToFloat(anyValue: Any): number {
  const doubleValue = DoubleValue.deserializeBinary(anyValue.getValue_asU8());
  return doubleValue.getValue();
}

//  integer to `Any`
export function Int32ToAny(value: number): Any {
  const int32Value = new Int32Value();
  int32Value.setValue(value);
  const serialized = int32Value.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.Int32Value');
}

//  `Any` to integer
export function AnyToInt32(anyValue: Any): number {
  const int32Value = Int32Value.deserializeBinary(anyValue.getValue_asU8());
  return int32Value.getValue();
}

//  boolean to `Any`
export function BoolToAny(value: boolean): Any {
  const boolValue = new BoolValue();
  boolValue.setValue(value);
  const serialized = boolValue.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.BoolValue');
}

//  `Any` to boolean
export function AnyToBool(anyValue: Any): boolean {
  const boolValue = BoolValue.deserializeBinary(anyValue.getValue_asU8());
  return boolValue.getValue();
}

//  bytes to `Any`
export function BytesToAny(value: Uint8Array): Any {
  const bytesValue = new BytesValue();
  bytesValue.setValue(value);
  const serialized = bytesValue.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.BytesValue');
}

//  JSON object to `Any`
export function JSONToAny(value: object): Any {
  const jsonString = JSON.stringify(value);
  const stringValue = new StringValue();
  stringValue.setValue(jsonString);
  const serialized = stringValue.serializeBinary();
  return pack(serialized, 'type.googleapis.com/google.protobuf.StringValue');
}

//  `Any` to JSON object
export function AnyToJSON(anyValue: Any): object {
  const stringValue = StringValue.deserializeBinary(anyValue.getValue_asU8());
  return JSON.parse(stringValue.getValue());
}

export function MapToObject(
  protoMap: Map<string, google_protobuf_any_pb.Any>,
): Record<string, any> {
  const result: Record<string, any> = {};
  protoMap.forEach((value, key) => {
    try {
      if (value instanceof Any) {
        // Get the underlying message as a plain object
        result[key] = new TextDecoder().decode(value.getValue_asU8());
      } else {
        result[key] = value;
      }
    } catch (x) {}
  });

  return result;
}

/**
 *
 * @param vl
 * @returns
 */
export function StringToListValue(vl: any[]): ListValue {
  const listValue = new ListValue();
  vl.forEach(obj => {
    listValue.addValues(Value.fromJavaScript(obj));
  });
  return listValue;
}
