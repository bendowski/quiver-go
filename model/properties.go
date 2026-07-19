package model

// Property keys used in Node.Properties. The parser (goparse) writes these
// keys, the schema DDL declares matching columns, and the dump package reads
// them back — schema's consistency test verifies the parser/schema pairing.
//
// PropID is special: parsed nodes carry their ID in Node.ID, and the store
// layer injects it as a property on insert.
const (
	PropID          = "id"
	PropName        = "name"
	PropImportPath  = "import_path"
	PropDir         = "dir"
	PropFilePath    = "file_path"
	PropPackageName = "package_name"
	PropSource      = "source"
	PropAlias       = "alias"
	PropPath        = "path"
	PropStartLine   = "start_line"
	PropEndLine     = "end_line"
	PropSignature   = "signature"
	PropReceiver    = "receiver"
	PropDocComment  = "doc_comment"
	PropExported    = "exported"
	PropKind        = "kind"
	PropTypeName    = "type_name"
	PropTag         = "tag"
)
