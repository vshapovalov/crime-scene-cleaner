package editor

const exportScript = `import json
import sys
from pathlib import Path

import UnityPy

bundle_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])
dictionary_path = Path(sys.argv[3]) if len(sys.argv) > 3 and sys.argv[3] else None

def read_rows(path):
    env = UnityPy.load(str(path))
    rows = []
    for obj in env.objects:
        if obj.type.name != "MonoBehaviour":
            continue
        try:
            tree = obj.read_typetree()
        except Exception:
            continue
        table_name = tree.get("m_Name")
        table_data = tree.get("m_TableData")
        if not table_name or not isinstance(table_data, list):
            continue
        for entry in table_data:
            rows.append({
                "table": table_name,
                "id": str(entry.get("m_Id", "")),
                "text": entry.get("m_Localized", ""),
            })
    return rows

rows = read_rows(bundle_path)
originals = {}
if dictionary_path and dictionary_path.exists():
    for row in read_rows(dictionary_path):
        originals[(row["table"], row["id"])] = row["text"]

for row in rows:
    row["original"] = originals.get((row["table"], row["id"]), row["text"])

output_path.write_text(json.dumps(rows, ensure_ascii=False, indent=2), encoding="utf-8")
print(json.dumps({"rows": len(rows)}, ensure_ascii=False))
`

const importScript = `import json
import shutil
import sys
from pathlib import Path

import UnityPy

bundle_path = Path(sys.argv[1])
rows_path = Path(sys.argv[2])
english_path = Path(sys.argv[3])
polish_path = Path(sys.argv[4])

rows = json.loads(rows_path.read_text(encoding="utf-8"))
translations = {
    (row["table"], str(row["id"])): row.get("text", "")
    for row in rows
}

env = UnityPy.load(str(bundle_path))
changed = 0

def apply_translations(env, locale_code=None, table_suffix=None):
    changed = 0
    for obj in env.objects:
        if obj.type.name != "MonoBehaviour":
            continue
        try:
            tree = obj.read_typetree()
        except Exception:
            continue
        table_name = tree.get("m_Name")
        table_data = tree.get("m_TableData")
        if not table_name or not isinstance(table_data, list):
            continue
        table_changed = False
        for entry in table_data:
            key = (table_name, str(entry.get("m_Id", "")))
            if key not in translations:
                continue
            new_text = translations[key]
            if entry.get("m_Localized", "") != new_text:
                entry["m_Localized"] = new_text
                changed += 1
                table_changed = True
        if locale_code is not None:
            locale_id = tree.get("m_LocaleId")
            if isinstance(locale_id, dict) and locale_id.get("m_Code") != locale_code:
                locale_id["m_Code"] = locale_code
                table_changed = True
        if table_suffix is not None:
            new_name = rewrite_table_suffix(table_name, table_suffix)
            if new_name != table_name:
                tree["m_Name"] = new_name
                table_changed = True
        if table_changed:
            obj.save_typetree(tree)
    return changed

def rewrite_table_suffix(name, suffix):
    for old_suffix in ("_ru", "_en", "_pl"):
        if name.endswith(old_suffix):
            return name[:-len(old_suffix)] + suffix
    return name + suffix

changed = apply_translations(env)
with bundle_path.open("wb") as out:
    out.write(env.file.save())

def export_for_locale(target_path, locale_code, table_suffix):
    shutil.copy2(bundle_path, target_path)
    locale_env = UnityPy.load(str(target_path))
    apply_translations(locale_env, locale_code=locale_code, table_suffix=table_suffix)
    with target_path.open("wb") as out:
        out.write(locale_env.file.save())

export_for_locale(english_path, "en", "_en")
export_for_locale(polish_path, "pl", "_pl")

print(json.dumps({"changed": changed, "exports": [str(english_path), str(polish_path)]}, ensure_ascii=False))
`
