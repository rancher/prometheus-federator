package project

// Copied from https://github.com/rancher/wrangler/blob/004e382969b42fb2f538ffd6699569d30e490428/pkg/data/merge.go#L3-L24
// Why did we copy the code? The logic for checking bothMaps needs to account for more possible types than map[string]interface{},
// namely v1alpha1.GenericMap and map[interface{}]interface{}

func MergeMaps(base, overlay map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		if baseMap, overlayMap, bothMaps := bothMaps(result[k], v); bothMaps {
			v = MergeMaps(baseMap, overlayMap)
		}
		result[k] = v
	}
	return result
}

func bothMaps(left, right interface{}) (map[string]interface{}, map[string]interface{}, bool) {
	leftMap, isMap := getMap(left)
	if !isMap {
		return nil, nil, false
	}
	rightMap, isMap := getMap(right)
	if !isMap {
		return nil, nil, false
	}
	return leftMap, rightMap, true
}

func getMap(entry interface{}) (map[string]interface{}, bool) {
	// check if map[string]interface{}
	entryMapStringInterface, isMapStringInterface := entry.(map[string]interface{})
	if isMapStringInterface {
		return entryMapStringInterface, true
	}

	// check if v1alpha1.GenericMap
	entryGenericMap, isGenericMap := entry.(map[string]interface{})
	if isGenericMap {
		return entryGenericMap, true
	}

	// check if map[interface{}]interface{}
	entryMapInterfaceInterface, isMapInterfaceInterface := entry.(map[interface{}]interface{})
	if isMapInterfaceInterface {
		return convertMapInterfaceInterfaceToMapStringInterface(entryMapInterfaceInterface)
	}

	return nil, false
}

func convertMapInterfaceInterfaceToMapStringInterface(entry map[interface{}]interface{}) (map[string]interface{}, bool) {
	out := make(map[string]interface{}, len(entry))
	for k, v := range entry {
		key, isString := k.(string)
		if !isString {
			return nil, false
		}
		out[key] = v
	}
	return out, true
}
