# The import ID is the provider's name. The password cannot be read back and
# is never compared, so the imported provider keeps the password SAP holds;
# password_wo in the configuration is sent only when the provider is created.
terraform import sapintegrationsuite_api_provider.backend ES5_1
