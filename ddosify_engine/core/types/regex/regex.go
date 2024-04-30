package regex

const DynamicVariableRegex = `\{{(_)[^}]+\}}`
const JsonDynamicVariableRegex = `\"{{(_)[^}]+\}}"`

const EnvironmentVariableRegex = `{{[a-zA-Z$][a-zA-Z0-9_().-]*}}`
const JsonEnvironmentVarRegex = `\"{{[a-zA-Z$][a-zA-Z0-9_().-]*}}"`
