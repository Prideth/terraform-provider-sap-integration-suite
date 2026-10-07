---
page_title: "sapintegrationsuite_api_management_virtual_host Resource - sapintegrationsuite"
subcategory: "API Management"
description: |-
  An additional virtual host of Classic API Management on the tenant's default domain, optionally
  with mutual TLS: a host name under which the API portal exposes API proxies.
---

# sapintegrationsuite_api_management_virtual_host (Resource)

A virtual host is a host name under which the API portal of Classic API Management serves API
proxies. Every tenant has a default one; additional hosts let you separate APIs by audience or
stage, for example `prod-apis.<tenant domain>` next to `test-apis.<tenant domain>`, without a
second tenant. This resource manages such additional hosts on the tenant's default domain: you
choose the alias, SAP appends the domain. A host can also demand a client certificate from every
caller (mutual TLS), for example one host for partners with certificates next to an ordinary one.

**Status:** partial. SAP Help documents the requests (*Configuring a Default Domain for a Virtual
Host*), and the resource's acceptance test created, renamed, imported and deleted a host on a
tenant. Mutual TLS follows SAP Help's *Configuring Mutual TLS for Default Domain Virtual Host*;
its acceptance test has not run on a tenant yet. Partial because virtual hosts with a custom
domain are not managed.

## Prerequisites

- The `api_management_self_service` block in the provider configuration. SAP lets only the role
  `APIManagement.SelfService.Administrator` change virtual hosts; the key of the
  `api_management` block (`APIPortal.Administrator`) is refused with HTTP 403. Create a second
  service instance of *API Management, API portal* with plan `apiportal-apiaccess` and the
  parameter `{"role": "APIManagement.SelfService.Administrator"}`, then a service key for it:

  ```terraform
  provider "sapintegrationsuite" {
    api_management_self_service {
      host          = var.api_portal_url             # the key's "url"
      token_url     = var.self_service_token_url     # the key's "tokenUrl"
      client_id     = var.self_service_client_id
      client_secret = var.self_service_client_secret
    }
  }
  ```

- The requests carry the subdomain of the subaccount (`accountId`). The provider takes it from
  the token URL, `https://<subdomain>.authentication.<region>.hana.ondemand.com/oauth/token`;
  set `subaccount_subdomain` in the block if your token URL looks different.

## Lifecycle

| Operation | What happens |
|---|---|
| Create | Sends a `CREATE` request with the alias to `Configuration.svc/VirtualHostRequests`, then reads the host from `Management.svc/VirtualHosts`. On the tested tenant the host existed as soon as SAP answered. |
| Read | Looks the host up by its ID in the list of virtual hosts: the self-service key may read the list but not a single host. A host deleted outside Terraform is removed from state and created again on the next apply. |
| Update | A changed `alias` renames the host in place with an `UPDATE` request; ID and port stay. SAP asks to redeploy and republish the API proxies of products that use the host afterwards. |
| Delete | Sends a `DELETE` request. SAP refuses it while the host is the default one or while an API proxy, a draft or a revision refers to it. A host already gone counts as deleted. |

`host_name` is `<alias>.<tenant domain>`, for example
`prod-apis.mysubaccount.apimanagement.eu10.hana.ondemand.com`. SAP Help shows
`prod-apis.sapdefaultdomain` in its sample answer; a tenant returned the real host name.

## Mutual TLS

With `client_auth_enabled = true`, the API gateway asks every caller of the host for a client
certificate during the TLS handshake and accepts only certificates the truststore in
`trust_store` vouches for. API proxies behind the host need no policy for this; a caller without
a valid certificate does not get through the handshake. The host's own server certificate stays
SAP's certificate for the default domain.

- **The truststore comes first.** Create it in the SAP Integration Suite UI (*Configure* >
  *APIs* > *Certificates* > *Create*, type *Trust Store*) and upload the client certificates as
  PEM or DER, or the certificates of the CA that issues them. SAP asks for the whole chain (client,
  intermediate and root certificates) in the truststore and, for certificates created or renewed
  for virtual hosts, the *Client Authentication* extended key usage (SAP Note 3725252). There is
  no documented API to create the truststore itself, so Terraform only refers to it.
- **`trust_store`** is the truststore's name, or `ref://<name>` for a
  [certificate store reference](api_management_certificate_store_reference.md) that points at it.
  A reference is the better choice for rotation: when the certificates change, create a new
  truststore in the UI and repoint the reference; the host itself does not change.
- **Switching it on or off** is an in-place update with the documented `UPDATE` request, which
  now carries `isClientAuthEnabled` and `trustStore`. Hosts without mutual TLS keep getting the
  request without those fields, exactly as before. As with a renamed host, SAP asks to redeploy
  and republish the API proxies of products that use the host.
- **Drift:** `client_auth_enabled` and `trust_store` are read back from the host list. A
  truststore SAP still lists after client authentication was switched off does not count.

## Example Usage

```terraform
# Needs the api_management_self_service block (a key with the role
# APIManagement.SelfService.Administrator) in the provider block.

# An additional host name for production APIs: prod-apis.<tenant domain>.
resource "sapintegrationsuite_api_management_virtual_host" "prod" {
  alias = "prod-apis"
}

# The full host name, for DNS documentation or API consumers.
output "prod_api_host" {
  value = sapintegrationsuite_api_management_virtual_host.prod.host_name
}

# A host for partners that must present a client certificate (mutual TLS).
# The truststore "partner-clients" holds their certificates or those of the
# CAs that issued them; it is created in the SAP Integration Suite UI
# (Configure, APIs, Certificates). The reference lets you rotate to a new
# truststore later without touching the host.
resource "sapintegrationsuite_api_management_certificate_store_reference" "partner_clients" {
  name                   = "partner-clients-current"
  certificate_store_name = "partner-clients"
}

resource "sapintegrationsuite_api_management_virtual_host" "partners" {
  alias               = "partner-apis"
  client_auth_enabled = true
  trust_store         = "ref://${sapintegrationsuite_api_management_certificate_store_reference.partner_clients.name}"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `alias` (String) The alias, the first label of the host name, for example prod-apis. Letters, digits and hyphens, not starting or ending with a hyphen, at most 63 characters, and unique on the tenant. Changing it renames the host in place; SAP then asks to redeploy and republish the API proxies of products that use it.

### Optional

- `client_auth_enabled` (Boolean) Mutual TLS: whether the host asks every client for a certificate and accepts only those trust_store vouches for. Needs trust_store. Switching it on or off changes the host in place; SAP then asks to redeploy and republish the API proxies of products that use it. Default false.
- `trust_store` (String) The truststore that holds the client certificates, or the certificates of the CAs that issued them, for client_auth_enabled: the name of a truststore of the API portal, or ref://<name> for a certificate store reference that points at one (see sapintegrationsuite_api_management_certificate_store_reference). SAP asks for the whole chain (client, intermediate and root certificates) in the truststore, and for client certificates with the Client Authentication extended key usage. The truststore itself is created in the SAP Integration Suite UI. Only allowed with client_auth_enabled.

### Read-Only

- `default` (Boolean) Whether this is the API portal's default virtual host. The provider creates additional hosts only; an imported default host keeps this flag on update.
- `host_name` (String) The full host name SAP assigned, <alias>.<tenant domain>.
- `id` (String) The virtual host's ID (virtualHostId), assigned by SAP.
- `port` (Number) The port of the virtual host, 443 on the tenant this was tested with.
- `ssl` (Boolean) Whether the virtual host serves HTTPS.

## Import

```shell
# The import ID is the virtual host's ID, its alias or its full host name.
terraform import sapintegrationsuite_api_management_virtual_host.prod prod-apis
```

The ID may be the virtual host's ID (`virtualHostId`, a UUID), its alias or its full host name.
Hosts with mutual TLS on the default domain are imported with their client authentication and
truststore; hosts with a custom domain or a keystore cannot be imported.

## Limitations

- **SAP:** an alias has at most 63 characters of letters, digits and hyphens, does not start or
  end with a hyphen, and is unique on the tenant; SAP refuses anything else with HTTP 400, and the
  provider checks the format while planning. Deletion is refused while proxies refer to the host.
- **SAP:** `Configuration.svc` does not serve its `$metadata` to the API portal keys, so the
  request fields are known from SAP Help and the tenant's answers only.
- **Provider:** only virtual hosts on the default domain are managed. Custom domains (keystore,
  key alias, custom domain flag) are documented by SAP but need DNS and a server certificate for a
  test, and the resource refuses to change or import a host that uses them.
- **Provider:** mutual TLS is built from SAP Help's documented requests; its acceptance test
  (`TestAccAPIManagementVirtualHost_mutualTLS`) has not run on a tenant yet. How SAP lists a
  `ref://` truststore and whether switching client authentication off clears the truststore are
  open until then.
- **Provider:** the resource never makes a host the default one; an imported default host keeps
  its flag when its alias changes.

## Related

- [`sapintegrationsuite_api_management_certificate_store_reference`](api_management_certificate_store_reference.md),
  the truststore reference a mutual TLS host can use in `trust_store`.
- [Classic API Management guide](../guides/classic-api-management.md).
