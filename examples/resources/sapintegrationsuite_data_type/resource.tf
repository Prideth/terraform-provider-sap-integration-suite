# Data types are unofficial: SAP documents no request for them. Needs
# enable_unofficial = true in the provider block.
resource "sapintegrationsuite_data_type" "order" {
  package_id   = sapintegrationsuite_integration_package.sales.id
  data_type_id = "Order"
  name         = "Order"
  namespace    = "urn:example:sales"
  description  = "Sales order header"
  xsd          = file("${path.module}/schemas/order.xsd")
}

# schemas/order.xsd:
#
# <xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
#             xmlns="urn:example:sales" targetNamespace="urn:example:sales">
#   <xsd:complexType name="Order">
#     <xsd:sequence>
#       <xsd:element name="OrderID" type="xsd:string"/>
#       <xsd:element name="Customer" type="xsd:string" minOccurs="0"/>
#     </xsd:sequence>
#   </xsd:complexType>
# </xsd:schema>
