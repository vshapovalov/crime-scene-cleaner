package patcher

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPatchAddressablesCatalogUpdatesSelectedBundleOnly(t *testing.T) {
	root := t.TempDir()
	gameDir, catalogPath := writeTestCatalog(t, root)
	sourceBundle := filepath.Join(root, "ukrainian-localization_en.bundle")
	if err := os.WriteFile(sourceBundle, []byte("translated english bundle"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := PatchAddressablesCatalog(gameDir, TargetEnglish, sourceBundle); err != nil {
		t.Fatalf("PatchAddressablesCatalog returned error: %v", err)
	}

	english := readCatalogOptions(t, catalogPath, testEnglishInternalID)
	if english.CRC != 0 {
		t.Fatalf("English CRC = %d, want 0", english.CRC)
	}
	if english.UseCRCForCachedBundles {
		t.Fatal("English UseCRCForCachedBundles = true, want false")
	}
	if english.BundleSize != 25 {
		t.Fatalf("English BundleSize = %d, want 25", english.BundleSize)
	}
	if english.Hash == "english-hash-00000000000000000000" {
		t.Fatal("English hash was not updated")
	}

	polish := readCatalogOptions(t, catalogPath, testPolishInternalID)
	if polish.CRC != 2222222222 || !polish.UseCRCForCachedBundles || polish.BundleSize != 222 {
		t.Fatalf("Polish entry was changed: %+v", polish)
	}
}

func TestPatchAddressablesCatalogAdjustsLaterDataIndexes(t *testing.T) {
	root := t.TempDir()
	gameDir, catalogPath := writeTestCatalog(t, root)
	sourceBundle := filepath.Join(root, "ukrainian-localization_en.bundle")
	if err := os.WriteFile(sourceBundle, bytes.Repeat([]byte("x"), 12345), 0o644); err != nil {
		t.Fatal(err)
	}

	before := readCatalogDataIndex(t, catalogPath, testPolishInternalID)
	if err := PatchAddressablesCatalog(gameDir, TargetEnglish, sourceBundle); err != nil {
		t.Fatalf("PatchAddressablesCatalog returned error: %v", err)
	}
	after := readCatalogDataIndex(t, catalogPath, testPolishInternalID)

	if after == before {
		t.Fatalf("Polish data index did not move; before=%d after=%d", before, after)
	}
	polish := readCatalogOptions(t, catalogPath, testPolishInternalID)
	if polish.BundleName != "polish-bundle" {
		t.Fatalf("Polish options could not be read after index adjustment: %+v", polish)
	}
}

const (
	testEnglishInternalID = "{UnityEngine.AddressableAssets.Addressables.RuntimePath}\\StandaloneWindows64\\localization-string-tables-english(en)_assets_all.bundle"
	testPolishInternalID  = "{UnityEngine.AddressableAssets.Addressables.RuntimePath}\\StandaloneWindows64\\localization-string-tables-polish(pl)_assets_all.bundle"
	testEnglishAssetID    = "{UnityEngine.AddressableAssets.Addressables.RuntimePath}\\StandaloneWindows64\\localization-asset-tables-english(en)_assets_all.bundle"
	testPolishAssetID     = "{UnityEngine.AddressableAssets.Addressables.RuntimePath}\\StandaloneWindows64\\localization-asset-tables-polish(pl)_assets_all.bundle"
)

type testBundleOptions struct {
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

func writeTestCatalog(t *testing.T, root string) (string, string) {
	t.Helper()

	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	catalogPath := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "catalog.json")
	if err := os.MkdirAll(filepath.Dir(catalogPath), 0o755); err != nil {
		t.Fatal(err)
	}

	english := testBundleOptions{
		Hash: "english-hash-00000000000000000000", CRC: 1111111111, BundleName: "english-bundle",
		BundleSize: 111, UseCRCForCachedBundles: true,
	}
	polish := testBundleOptions{
		Hash: "polish-hash-000000000000000000000", CRC: 2222222222, BundleName: "polish-bundle",
		BundleSize: 222, UseCRCForCachedBundles: true,
	}
	englishAsset := testBundleOptions{
		Hash: "english-asset-hash-00000000000000", CRC: 3333333333, BundleName: "english-asset-bundle",
		BundleSize: 333, UseCRCForCachedBundles: true,
	}
	polishAsset := testBundleOptions{
		Hash: "polish-asset-hash-000000000000000", CRC: 4000000000, BundleName: "polish-asset-bundle",
		BundleSize: 444, UseCRCForCachedBundles: true,
	}
	objects := [][]byte{
		encodeTestCatalogObject(t, english),
		encodeTestCatalogObject(t, polish),
		encodeTestCatalogObject(t, englishAsset),
		encodeTestCatalogObject(t, polishAsset),
	}
	extraData := bytes.Join(objects, nil)

	var entries bytes.Buffer
	writeInt32(t, &entries, 4)
	dataIndex := int32(0)
	for i, object := range objects {
		writeEntry(t, &entries, int32(i), 0, dataIndex)
		dataIndex += int32(len(object))
	}

	catalog := map[string]any{
		"m_InternalIds":        []string{testEnglishInternalID, testPolishInternalID, testEnglishAssetID, testPolishAssetID},
		"m_EntryDataString":    base64.StdEncoding.EncodeToString(entries.Bytes()),
		"m_ExtraDataString":    base64.StdEncoding.EncodeToString(extraData),
		"m_ProviderIds":        []string{},
		"m_resourceTypes":      []string{},
		"m_InternalIdPrefixes": []string{},
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalogPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return gameDir, catalogPath
}

func encodeTestCatalogObject(t *testing.T, options testBundleOptions) []byte {
	t.Helper()
	jsonData, err := json.Marshal(options)
	if err != nil {
		t.Fatal(err)
	}
	jsonUTF16 := utf16LEFromString(string(jsonData))
	assembly := []byte("Unity.ResourceManager")
	class := []byte("UnityEngine.ResourceManagement.ResourceProviders.AssetBundleRequestOptions")

	var out bytes.Buffer
	out.WriteByte(7)
	out.WriteByte(byte(len(assembly)))
	out.Write(assembly)
	out.WriteByte(byte(len(class)))
	out.Write(class)
	writeInt32(t, &out, int32(len(jsonUTF16)))
	out.Write(jsonUTF16)
	return out.Bytes()
}

func writeEntry(t *testing.T, out *bytes.Buffer, internalID int32, provider int32, dataIndex int32) {
	t.Helper()
	values := []int32{internalID, provider, 0, 0, dataIndex, internalID, 0}
	for _, value := range values {
		writeInt32(t, out, value)
	}
}

func writeInt32(t *testing.T, out *bytes.Buffer, value int32) {
	t.Helper()
	if err := binary.Write(out, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

func readCatalogOptions(t *testing.T, catalogPath string, internalID string) testBundleOptions {
	t.Helper()
	raw := readCatalogObjectJSON(t, catalogPath, internalID)
	var options testBundleOptions
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		t.Fatalf("unmarshal options: %v\n%s", err, raw)
	}
	return options
}

func readCatalogObjectJSON(t *testing.T, catalogPath string, internalID string) string {
	t.Helper()
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		InternalIDs     []string `json:"m_InternalIds"`
		EntryDataString string   `json:"m_EntryDataString"`
		ExtraDataString string   `json:"m_ExtraDataString"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	internalIndex := -1
	for i, id := range catalog.InternalIDs {
		if id == internalID {
			internalIndex = i
			break
		}
	}
	if internalIndex < 0 {
		t.Fatalf("internal ID %q not found", internalID)
	}
	entryData, err := base64.StdEncoding.DecodeString(catalog.EntryDataString)
	if err != nil {
		t.Fatal(err)
	}
	dataIndex := findTestDataIndex(t, entryData, int32(internalIndex))
	extraData, err := base64.StdEncoding.DecodeString(catalog.ExtraDataString)
	if err != nil {
		t.Fatal(err)
	}
	object, err := parseCatalogJSONObject(extraData, dataIndex)
	if err != nil {
		t.Fatal(err)
	}
	return object.JSON
}

func readCatalogDataIndex(t *testing.T, catalogPath string, internalID string) int32 {
	t.Helper()
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		InternalIDs     []string `json:"m_InternalIds"`
		EntryDataString string   `json:"m_EntryDataString"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	internalIndex := -1
	for i, id := range catalog.InternalIDs {
		if id == internalID {
			internalIndex = i
			break
		}
	}
	entryData, err := base64.StdEncoding.DecodeString(catalog.EntryDataString)
	if err != nil {
		t.Fatal(err)
	}
	return findTestDataIndex(t, entryData, int32(internalIndex))
}

func findTestDataIndex(t *testing.T, entryData []byte, internalIndex int32) int32 {
	t.Helper()
	count := int(binary.LittleEndian.Uint32(entryData[:4]))
	for i := 0; i < count; i++ {
		offset := 4 + i*28
		if int32(binary.LittleEndian.Uint32(entryData[offset:offset+4])) == internalIndex {
			return int32(binary.LittleEndian.Uint32(entryData[offset+16 : offset+20]))
		}
	}
	t.Fatalf("entry for internal index %d not found", internalIndex)
	return 0
}
