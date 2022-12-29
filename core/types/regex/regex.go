package regex

const DynamicVariableRegex = `\{{(_)[^}]+\}}`
const JsonDynamicVariableRegex = `\"{{(_)[^}]+\}}"`

const EnvironmentVariableRegex = `\{{[^_]\w*\}}`
const JsonEnvironmentVarRegex = `\"{{[^_]\w*\}}"`
