@provision-various-versions @core
Feature: Provision all major OpenShift versions
  As a user I can provision major versions of OpenShift

  Scenario Outline: Provision all major OpenShift versions
    Given Minishift has state "Does Not Exist"
      And image caching is disabled
     # TODO: Replace --ocp-tag with --openshift-version in v3.8
     When executing "minishift start --ocp-tag <serverVersion>" succeeds
     Then Minishift should have state "Running"
     When executing "minishift openshift version" succeeds
     Then stdout should contain
      """
      openshift <serverVersion>
      """
      And JSON config file ".minishift/machines/minishift-state.json" contains key "OcPath" with value matching "<ocVersion>"
     When executing "minishift oc-env" succeeds
     Then stdout should contain
      """
      <ocVersion>
      """
      And "status code" of HTTP request to "/healthz" of OpenShift instance is equal to "200"
      And "body" of HTTP request to "/healthz" of OpenShift instance contains "ok"
      And with up to "10" retries with wait period of "2s" the "status code" of HTTP request to "/console" of OpenShift instance is equal to "200"
      And "body" of HTTP request to "/console" of OpenShift instance contains "<title>OpenShift Web Console</title>"
     When executing "minishift delete --force" succeeds
     Then Minishift should have state "Does Not Exist"

  Examples:
    | serverVersion | ocVersion |
    | v3.10.45      | v3.10.45  |

  Scenario: Provision latest OpenShift version
    Given Minishift has state "Does Not Exist"
      And image caching is disabled
     When executing "minishift start --openshift-version latest" succeeds
     Then Minishift should have state "Running"
     When executing "minishift delete --force" succeeds
     Then Minishift should have state "Does Not Exist"
