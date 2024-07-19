# Using JQ

## Filter results using select

```shell
cat /tmp/a | jq  -r '.[] | select (.detailed_merge_status != "need_rebase") | select (.title == "Updating Backstage catalog file")'
```

## Format output

use `( .value.to.use )` then a `+` and a `" string to output "`.
If values from JQ are not text, just use `( .value.to.use | @text)`

```shell
cat /tmp/a | jq  -r '.[] | select (.detailed_merge_status == "need_rebase") | select (.title == "Updating Backstage catalog file") | "https://gitlab.domain.net/api/v4/projects/" + (.project_id | @text) + "/merge_requests/" + (.iid | @text) + "/rebase"'
```