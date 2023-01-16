package regex

const DynamicVariableRegex = `\{{(_)[^}]+\}}`
const JsonDynamicVariableRegex = `\"{{(_)[^}]+\}}"`

const EnvironmentVariableRegex = `\{{[^_][a-zA-Z0-9_().]*\}}`
const JsonEnvironmentVarRegex = `\"{{[^_][a-zA-Z0-9_().]*\}}"`
