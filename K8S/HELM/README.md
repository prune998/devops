# Helm

## Make any chart debug the computed values

Warning: this will not compute the yamls

```shell
mkdir templates && cd templates

cat >> _helper.yaml <<EOF
{{/*
Dump a variable for debugging purposes.
Usage: {{ include "chart.var_dump" .Values.someVar }}
*/}}
{{- define "chart.var_dump" -}}
{{- $msg := printf "DEBUG DUMP: %s" (. | mustToPrettyJson) -}}
{{- fail $msg -}}
{{- end -}}
EOF

cat >> debug.yaml <<EOF
{{ include "chart.var_dump" .Values }}
EOF
```

Then `helm template ...` to get a dump of the final chart values
