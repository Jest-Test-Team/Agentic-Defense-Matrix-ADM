# Shown under "Changes to Outputs" during plan. OCIDs are truncated because CI
# logs are public; match the suffix against the OCI console to get the full OCID.
output "network_diagnostics" {
  description = "vcn-count quota usage and VCNs/subnets across all compartments"
  value = {
    vcn_limit_used      = data.oci_limits_resource_availability.vcn_count.used
    vcn_limit_available = data.oci_limits_resource_availability.vcn_count.available
    compartments_searched = [
      for key, c in local.compartments : c.name
    ]
    vcns = [
      for v in local.all_vcns : {
        name        = v.name
        state       = v.state
        compartment = v.compartment
        cidr_blocks = v.cidr_blocks
        id_suffix   = substr(v.id, -12, 12)
        subnets = [
          for s in data.oci_core_subnets.by_vcn[v.id].subnets : {
            name       = s.display_name
            cidr_block = s.cidr_block
            public     = !s.prohibit_public_ip_on_vnic
            id_suffix  = substr(s.id, -12, 12)
          }
        ]
      }
    ]
  }
}

output "instance_public_ip" {
  description = "Public IP of the ADM instance"
  value       = oci_core_instance.adm_instance.public_ip
}

output "instance_private_ip" {
  description = "Private IP of the ADM instance"
  value       = oci_core_instance.adm_instance.private_ip
}

output "instance_id" {
  description = "OCID of the ADM instance"
  value       = oci_core_instance.adm_instance.id
}

output "volume_id" {
  description = "OCID of the data volume"
  value       = oci_core_volume.adm_volume.id
}

output "ssh_command" {
  description = "SSH command to connect to the instance"
  value       = "ssh -i ~/.ssh/adm_key ubuntu@${oci_core_instance.adm_instance.public_ip}"
}

output "gateway_url" {
  description = "Gateway URL"
  value       = "http://${oci_core_instance.adm_instance.public_ip}:8080"
}

output "ollama_url" {
  description = "Ollama URL"
  value       = "http://${oci_core_instance.adm_instance.public_ip}:11434"
}

output "next_steps" {
  description = "Next steps after deployment"
  value       = <<-EOT
    1. SSH into the instance:
       ssh -i ~/.ssh/adm_key ubuntu@${oci_core_instance.adm_instance.public_ip}

    2. Check deployment status:
       sudo -u adm /opt/adm/scripts/status.sh

    3. View logs:
       sudo -u adm /opt/adm/scripts/logs.sh

    4. Access the Gateway:
       http://${oci_core_instance.adm_instance.public_ip}:8080/v1/health
  EOT
}
