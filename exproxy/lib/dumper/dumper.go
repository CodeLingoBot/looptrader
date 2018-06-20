package dumper

import "encoding/json"

func DumpObject(obj interface{}) []byte {
	if obj == nil {
		return []byte{}
	}
	data, _ := json.Marshal(obj)
	return data
}

func DumpObjectIndent(obj interface{}) []byte {
	if obj == nil {
		return []byte{}
	}
	data, _ := json.MarshalIndent(obj, "", "  ")
	return data
}
