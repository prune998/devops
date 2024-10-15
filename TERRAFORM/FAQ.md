# Terraform,Tofu and Terragrunt

## Adding debug to TF

```bash
TF_LOG=debug terraform plan
```

## adding debug to TG

```bash
terragrunt run-all plan --terragrunt-log-level debug --terragrunt-debug
```