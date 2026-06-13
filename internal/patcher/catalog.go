package patcher

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf16"
)

const catalogRelativePath = "CrimeCleaner_Data/StreamingAssets/aa/catalog.json"

type catalogJSONObject struct {
	Header []byte
	JSON   string
	Start  int32
	End    int32
}

type assetBundleRequestOptions struct {
	Hash                             string `json:"m_Hash"`
	CRC                              uint32 `json:"m_Crc"`
	Timeout                          int    `json:"m_Timeout"`
	ChunkedTransfer                  bool   `json:"m_ChunkedTransfer"`
	RedirectLimit                    int    `json:"m_RedirectLimit"`
	RetryCount                       int    `json:"m_RetryCount"`
	BundleName                       string `json:"m_BundleName"`
	AssetLoadMode                    int    `json:"m_AssetLoadMode"`
	BundleSize                       int64  `json:"m_BundleSize"`
	UseCRCForCachedBundles           bool   `json:"m_UseCrcForCachedBundles"`
	UseUWRForLocalBundles            bool   `json:"m_UseUWRForLocalBundles"`
	ClearOtherCachedVersionsWhenLoad bool   `json:"m_ClearOtherCachedVersionsWhenLoaded"`
}

func CatalogPath(gameDir string) string {
	return filepath.Join(gameDir, filepath.FromSlash(catalogRelativePath))
}

func PatchAddressablesCatalog(gameDir string, target TargetLanguage, sourceBundle string) error {
	targetPath, err := TargetBundlePath(gameDir, target)
	if err != nil {
		return err
	}
	catalogPath := CatalogPath(gameDir)
	sourceStat, err := os.Stat(sourceBundle)
	if err != nil {
		return fmt.Errorf("бандл перекладу відсутній: %w", err)
	}
	if sourceStat.IsDir() {
		return fmt.Errorf("%s is a directory", sourceBundle)
	}
	sourceBytes, err := os.ReadFile(sourceBundle)
	if err != nil {
		return fmt.Errorf("читання бандла перекладу: %w", err)
	}
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("catalog.json відсутній: %w", err)
	}

	patched, err := patchCatalogBytes(catalogBytes, addressablesInternalID(targetPath), sourceBytes, sourceStat.Size())
	if err != nil {
		return err
	}
	return os.WriteFile(catalogPath, patched, 0o644)
}

func patchCatalogBytes(catalogBytes []byte, targetInternalID string, sourceBytes []byte, sourceSize int64) ([]byte, error) {
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(catalogBytes, &catalog); err != nil {
		return nil, fmt.Errorf("читання catalog.json: %w", err)
	}

	var internalIDs []string
	if err := json.Unmarshal(catalog["m_InternalIds"], &internalIDs); err != nil {
		return nil, fmt.Errorf("читання m_InternalIds: %w", err)
	}
	internalIDIndex := -1
	for i, id := range internalIDs {
		if id == targetInternalID {
			internalIDIndex = i
			break
		}
	}
	if internalIDIndex < 0 {
		return nil, fmt.Errorf("запис бандла не знайдено у catalog.json: %s", targetInternalID)
	}

	var entryDataBase64 string
	if err := json.Unmarshal(catalog["m_EntryDataString"], &entryDataBase64); err != nil {
		return nil, fmt.Errorf("читання m_EntryDataString: %w", err)
	}
	entryData, err := base64.StdEncoding.DecodeString(entryDataBase64)
	if err != nil {
		return nil, fmt.Errorf("декодування m_EntryDataString: %w", err)
	}
	dataIndex, err := findCatalogDataIndex(entryData, int32(internalIDIndex))
	if err != nil {
		return nil, err
	}

	var extraDataBase64 string
	if err := json.Unmarshal(catalog["m_ExtraDataString"], &extraDataBase64); err != nil {
		return nil, fmt.Errorf("читання m_ExtraDataString: %w", err)
	}
	extraData, err := base64.StdEncoding.DecodeString(extraDataBase64)
	if err != nil {
		return nil, fmt.Errorf("декодування m_ExtraDataString: %w", err)
	}
	object, err := parseCatalogJSONObject(extraData, dataIndex)
	if err != nil {
		return nil, err
	}

	var options assetBundleRequestOptions
	if err := json.Unmarshal([]byte(object.JSON), &options); err != nil {
		return nil, fmt.Errorf("читання AssetBundleRequestOptions: %w", err)
	}
	hash := md5.Sum(sourceBytes)
	options.Hash = hex.EncodeToString(hash[:])
	options.CRC = 0
	options.BundleSize = sourceSize
	options.UseCRCForCachedBundles = false

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("запис AssetBundleRequestOptions: %w", err)
	}
	newJSONBytes := utf16LEFromString(string(optionsJSON))
	newObject := make([]byte, 0, len(object.Header)+4+len(newJSONBytes))
	newObject = append(newObject, object.Header...)
	newObject = binary.LittleEndian.AppendUint32(newObject, uint32(len(newJSONBytes)))
	newObject = append(newObject, newJSONBytes...)

	newExtraData := make([]byte, 0, len(extraData)+len(newObject)-int(object.End-object.Start))
	newExtraData = append(newExtraData, extraData[:object.Start]...)
	newExtraData = append(newExtraData, newObject...)
	newExtraData = append(newExtraData, extraData[object.End:]...)

	delta := int32(len(newObject)) - (object.End - object.Start)
	if delta != 0 {
		adjustCatalogDataIndexes(entryData, dataIndex, delta)
	}

	entryDataEncoded, err := json.Marshal(base64.StdEncoding.EncodeToString(entryData))
	if err != nil {
		return nil, err
	}
	extraDataEncoded, err := json.Marshal(base64.StdEncoding.EncodeToString(newExtraData))
	if err != nil {
		return nil, err
	}
	catalog["m_EntryDataString"] = entryDataEncoded
	catalog["m_ExtraDataString"] = extraDataEncoded

	patched, err := json.Marshal(catalog)
	if err != nil {
		return nil, fmt.Errorf("запис catalog.json: %w", err)
	}
	return patched, nil
}

func addressablesInternalID(targetPath string) string {
	return `{UnityEngine.AddressableAssets.Addressables.RuntimePath}\StandaloneWindows64\` + filepath.Base(targetPath)
}

func findCatalogDataIndex(entryData []byte, internalIDIndex int32) (int32, error) {
	if len(entryData) < 4 {
		return 0, fmt.Errorf("m_EntryDataString пошкоджений")
	}
	count := int(int32(binary.LittleEndian.Uint32(entryData[:4])))
	for i := 0; i < count; i++ {
		offset := 4 + i*28
		if offset+28 > len(entryData) {
			return 0, fmt.Errorf("m_EntryDataString має неправильний розмір")
		}
		internalID := int32(binary.LittleEndian.Uint32(entryData[offset : offset+4]))
		if internalID == internalIDIndex {
			return int32(binary.LittleEndian.Uint32(entryData[offset+16 : offset+20])), nil
		}
	}
	return 0, fmt.Errorf("запис бандла не знайдено у m_EntryDataString")
}

func adjustCatalogDataIndexes(entryData []byte, changedDataIndex int32, delta int32) {
	count := int(int32(binary.LittleEndian.Uint32(entryData[:4])))
	for i := 0; i < count; i++ {
		offset := 4 + i*28
		dataIndexOffset := offset + 16
		dataIndex := int32(binary.LittleEndian.Uint32(entryData[dataIndexOffset : dataIndexOffset+4]))
		if dataIndex > changedDataIndex {
			binary.LittleEndian.PutUint32(entryData[dataIndexOffset:dataIndexOffset+4], uint32(dataIndex+delta))
		}
	}
}

func parseCatalogJSONObject(extraData []byte, offset int32) (catalogJSONObject, error) {
	if offset < 0 || int(offset) >= len(extraData) {
		return catalogJSONObject{}, fmt.Errorf("індекс m_ExtraDataString поза межами")
	}
	start := int(offset)
	if extraData[start] != 7 {
		return catalogJSONObject{}, fmt.Errorf("очікував JsonObject у m_ExtraDataString")
	}
	cursor := start + 1
	if cursor >= len(extraData) {
		return catalogJSONObject{}, fmt.Errorf("JsonObject пошкоджений")
	}
	assemblyLength := int(extraData[cursor])
	cursor++
	cursor += assemblyLength
	if cursor >= len(extraData) {
		return catalogJSONObject{}, fmt.Errorf("JsonObject має неправильну довжину assembly")
	}
	classLength := int(extraData[cursor])
	cursor++
	cursor += classLength
	if cursor+4 > len(extraData) {
		return catalogJSONObject{}, fmt.Errorf("JsonObject має неправильну довжину class")
	}
	headerEnd := cursor
	jsonByteLength := int(int32(binary.LittleEndian.Uint32(extraData[cursor : cursor+4])))
	cursor += 4
	jsonEnd := cursor + jsonByteLength
	if jsonByteLength < 0 || jsonEnd > len(extraData) {
		return catalogJSONObject{}, fmt.Errorf("JsonObject має неправильну довжину JSON")
	}
	return catalogJSONObject{
		Header: append([]byte(nil), extraData[start:headerEnd]...),
		JSON:   stringFromUTF16LE(extraData[cursor:jsonEnd]),
		Start:  int32(start),
		End:    int32(jsonEnd),
	}, nil
}

func utf16LEFromString(value string) []byte {
	encoded := utf16.Encode([]rune(value))
	out := make([]byte, len(encoded)*2)
	for i, word := range encoded {
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], word)
	}
	return out
}

func stringFromUTF16LE(data []byte) string {
	words := make([]uint16, len(data)/2)
	for i := range words {
		words[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
	}
	return string(utf16.Decode(words))
}
