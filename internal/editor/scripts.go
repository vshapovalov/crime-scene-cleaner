package editor

const exportScript = `import json
import sys
from pathlib import Path

import UnityPy

bundle_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])

env = UnityPy.load(str(bundle_path))
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

output_path.write_text(json.dumps(rows, ensure_ascii=False, indent=2), encoding="utf-8")
print(json.dumps({"rows": len(rows)}, ensure_ascii=False))
`

const importScript = `import json
import sys
from pathlib import Path

import UnityPy

bundle_path = Path(sys.argv[1])
rows_path = Path(sys.argv[2])

rows = json.loads(rows_path.read_text(encoding="utf-8"))
translations = {
    (row["table"], str(row["id"])): row.get("text", "")
    for row in rows
}

env = UnityPy.load(str(bundle_path))
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
    if table_changed:
        obj.save_typetree(tree)

if changed:
    with bundle_path.open("wb") as out:
        out.write(env.file.save())

print(json.dumps({"changed": changed}, ensure_ascii=False))
`
