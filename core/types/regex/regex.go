package regex

const DynamicVariableRegex = `\{{(_)[^}]+\}}`
const EnvironmentVariableRegex = `\{{[^_]\w*\}}`
