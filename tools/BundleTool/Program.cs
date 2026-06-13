using System.Text.Json;
using System.Text.Json.Serialization;
using AssetsTools.NET;
using AssetsTools.NET.Extra;

if (args.Length < 1)
{
    Usage();
    return 2;
}

try
{
    return args[0] switch
    {
        "export" when args.Length == 3 => Export(args[1], args[2]),
        "import" when args.Length == 4 => Import(args[1], args[2], args[3], null, null),
        "import" when args.Length == 6 => Import(args[1], args[2], args[3], args[4], args[5]),
        "inspect" when args.Length == 3 => Inspect(args[1], args[2]),
        "roundtrip" when args.Length == 3 => RoundTrip(args[1], args[2]),
        _ => Usage(),
    };
}
catch (Exception ex)
{
    Console.Error.WriteLine(ex);
    return 1;
}

static int Usage()
{
    Console.Error.WriteLine("Usage:");
    Console.Error.WriteLine("  BundleTool export <bundle> <rows.json>");
    Console.Error.WriteLine("  BundleTool import <template.bundle> <rows.json> <output.bundle> [locale-code table-suffix]");
    Console.Error.WriteLine("  BundleTool inspect <bundle> <metadata.json>");
    Console.Error.WriteLine("  BundleTool roundtrip <input.bundle> <output.bundle>");
    return 2;
}

static int Export(string bundlePath, string outputPath)
{
    using var loaded = LoadedBundle.Open(bundlePath);
    var rows = new List<TranslationRow>();

    foreach (var info in loaded.Assets.file.GetAssetsOfType(AssetClassID.MonoBehaviour))
    {
        var baseField = loaded.Manager.GetBaseField(loaded.Assets, info);
        var tableName = baseField["m_Name"].AsString;
        var tableData = baseField["m_TableData.Array"];
        if (string.IsNullOrWhiteSpace(tableName) || tableData.IsDummy)
        {
            continue;
        }

        foreach (var entry in tableData.Children)
        {
            rows.Add(new TranslationRow
            {
                Table = tableName,
                Id = entry["m_Id"].AsLong.ToString(),
                Text = entry["m_Localized"].AsString,
            });
        }
    }

    WriteJson(outputPath, rows);
    Console.WriteLine(JsonSerializer.Serialize(new { rows = rows.Count }));
    return 0;
}

static int Import(string templatePath, string rowsPath, string outputPath, string? localeCode, string? tableSuffix)
{
    var rows = ReadJson<List<TranslationRow>>(rowsPath) ?? [];
    var translations = rows.ToDictionary(
        row => RowKey(row.Table, row.Id),
        row => row.Text ?? string.Empty);

    using var loaded = LoadedBundle.Open(templatePath);
    var changed = 0;

    foreach (var info in loaded.Assets.file.GetAssetsOfType(AssetClassID.MonoBehaviour))
    {
        var baseField = loaded.Manager.GetBaseField(loaded.Assets, info);
        var tableName = baseField["m_Name"].AsString;
        var tableData = baseField["m_TableData.Array"];
        if (string.IsNullOrWhiteSpace(tableName) || tableData.IsDummy)
        {
            continue;
        }

        var tableChanged = false;
        foreach (var entry in tableData.Children)
        {
            var id = entry["m_Id"].AsLong.ToString();
            if (!translations.TryGetValue(RowKey(tableName, id), out var text))
            {
                continue;
            }
            if (entry["m_Localized"].AsString != text)
            {
                entry["m_Localized"].AsString = text;
                changed++;
                tableChanged = true;
            }
        }

        if (!string.IsNullOrWhiteSpace(localeCode))
        {
            var locale = baseField["m_LocaleId.m_Code"];
            if (!locale.IsDummy && locale.AsString != localeCode)
            {
                locale.AsString = localeCode;
                tableChanged = true;
            }
        }

        if (!string.IsNullOrWhiteSpace(tableSuffix))
        {
            var newName = RewriteTableSuffix(tableName, tableSuffix);
            if (newName != tableName)
            {
                baseField["m_Name"].AsString = newName;
                tableChanged = true;
            }
        }

        if (tableChanged)
        {
            info.SetNewData(baseField);
        }
    }

    WritePackedBundle(loaded.Bundle.file, loaded.Assets.file, outputPath);
    Console.WriteLine(JsonSerializer.Serialize(new { changed, output = outputPath }));
    return 0;
}

static int Inspect(string bundlePath, string outputPath)
{
    using var loaded = LoadedBundle.Open(bundlePath);
    var tables = new List<TableInfo>();
    foreach (var info in loaded.Assets.file.GetAssetsOfType(AssetClassID.MonoBehaviour))
    {
        var baseField = loaded.Manager.GetBaseField(loaded.Assets, info);
        var tableName = baseField["m_Name"].AsString;
        var tableData = baseField["m_TableData.Array"];
        if (string.IsNullOrWhiteSpace(tableName) || tableData.IsDummy)
        {
            continue;
        }
        var locale = baseField["m_LocaleId.m_Code"];
        tables.Add(new TableInfo(tableName, locale.IsDummy ? string.Empty : locale.AsString));
    }

    var directories = loaded.Bundle.file.BlockAndDirInfo.DirectoryInfos
        .Select(info => info.Name)
        .ToArray();
    WriteJson(outputPath, new { tables, directories });
    return 0;
}

static int RoundTrip(string inputPath, string outputPath)
{
    using var loaded = LoadedBundle.Open(inputPath);
    WritePackedBundle(loaded.Bundle.file, loaded.Assets.file, outputPath);
    return 0;
}

static void WritePackedBundle(AssetBundleFile bundle, AssetsFile assets, string outputPath)
{
    bundle.BlockAndDirInfo.DirectoryInfos[0].SetNewData(assets);

    var tempPath = Path.Combine(Path.GetDirectoryName(outputPath) ?? ".", "." + Path.GetFileName(outputPath) + ".uncompressed");
    Directory.CreateDirectory(Path.GetDirectoryName(outputPath) ?? ".");

    using (var writer = new AssetsFileWriter(tempPath))
    {
        bundle.Write(writer);
    }

    using (var uncompressedReader = new AssetsFileReader(tempPath))
    {
        var uncompressedBundle = new AssetBundleFile();
        uncompressedBundle.Read(uncompressedReader);
        using var writer = new AssetsFileWriter(outputPath);
        uncompressedBundle.Pack(writer, AssetBundleCompressionType.LZ4);
    }
    File.Delete(tempPath);
}

static string RewriteTableSuffix(string value, string suffix)
{
    foreach (var existingSuffix in new[] { "_ru", "_en", "_pl" })
    {
        if (value.EndsWith(existingSuffix, StringComparison.Ordinal))
        {
            return value[..^existingSuffix.Length] + suffix;
        }

        var assetSuffix = existingSuffix + ".asset";
        if (value.EndsWith(assetSuffix, StringComparison.Ordinal))
        {
            return value[..^assetSuffix.Length] + suffix + ".asset";
        }
    }
    return value + suffix;
}

static string RowKey(string table, string id)
{
    return NormalizeTableName(table) + "\0" + id;
}

static string NormalizeTableName(string table)
{
    foreach (var suffix in new[] { "_ru", "_en", "_pl" })
    {
        if (table.EndsWith(suffix, StringComparison.Ordinal))
        {
            return table[..^suffix.Length];
        }
    }
    return table;
}

static T? ReadJson<T>(string path)
{
    return JsonSerializer.Deserialize<T>(File.ReadAllText(path), JsonOptions());
}

static void WriteJson<T>(string path, T value)
{
    Directory.CreateDirectory(Path.GetDirectoryName(path) ?? ".");
    File.WriteAllText(path, JsonSerializer.Serialize(value, JsonOptions()));
}

static JsonSerializerOptions JsonOptions()
{
    return new JsonSerializerOptions
    {
        WriteIndented = true,
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        Encoder = System.Text.Encodings.Web.JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
    };
}

sealed class LoadedBundle : IDisposable
{
    public AssetsManager Manager { get; }
    public BundleFileInstance Bundle { get; }
    public AssetsFileInstance Assets { get; }

    private LoadedBundle(AssetsManager manager, BundleFileInstance bundle, AssetsFileInstance assets)
    {
        Manager = manager;
        Bundle = bundle;
        Assets = assets;
    }

    public static LoadedBundle Open(string bundlePath)
    {
        var manager = new AssetsManager();
        manager.LoadClassPackage(Path.Combine(AppContext.BaseDirectory, "classdata.tpk"));
        var bundle = manager.LoadBundleFile(bundlePath, true);
        var assets = manager.LoadAssetsFileFromBundle(bundle, 0, false);
        return new LoadedBundle(manager, bundle, assets);
    }

    public void Dispose()
    {
        Manager.UnloadAll();
    }
}

sealed class TranslationRow
{
    [JsonPropertyName("table")]
    public string Table { get; set; } = string.Empty;

    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;

    [JsonPropertyName("original")]
    public string Original { get; set; } = string.Empty;
}

sealed record TableInfo(string Name, string Locale);
