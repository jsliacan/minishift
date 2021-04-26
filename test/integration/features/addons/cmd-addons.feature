@cmd-addons @core
Feature: Addons command and its subcommands

  @minishift-only @quick
  Scenario: Default add-ons are installed but not enabled by default
     Note: default add-ons were installed during previous commands automatically.
     When executing "minishift addons list"
     Then stdout should match "admin-user\s*: disabled\s*P\(0\)"
      And stdout should match "anyuid\s*: disabled\s*P\(0\)"
      And stdout should match "registry-route\s*: disabled\s*P\(0\)"
      And stdout should match "xpaas\s*: disabled\s*P\(0\)"
      And stdout should match "che\s*: disabled\s*P\(0\)"
      And stdout should match "htpasswd-identity-provider\s*: disabled\s*P\(0\)"
      And stdout should match "admissions-webhook\s*: disabled\s*P\(0\)"

  @minishift-only @quick
  Scenario: Verbose listing of add-ons installed by default
     When executing "minishift addons list --verbose"
     Then stdout should match "Name\s*: admin-user"
      And stdout should match "Description\s*: Create admin user and assign the cluster-admin role to it."
      And stdout should match "Enabled\s*: false"
      And stdout should match "Priority\s*: 0"
      And stdout should match "Url\s*:"
      And stdout should match "Openshift Version\s*:"
      And stdout should match "Name\s*: anyuid"
      And stdout should match "Description\s*: Changes the default security context constraints to allow pods to run as any user."
      And stdout should match "Enabled\s*: false"
      And stdout should match "Priority\s*: 0"
      And stdout should match "Url\s*:"
      And stdout should match "Openshift Version\s*:"
      And stdout should match "Name\s*: registry-route"
      And stdout should match "Description\s*: Create an edge terminated route for the OpenShift registry"
      And stdout should match "Enabled\s*: false"
      And stdout should match "Priority\s*: 0"
      And stdout should match "Url\s*:"
      And stdout should match "Openshift Version\s*:"
      And stdout should match "Name\s*: xpaas"
      And stdout should match "Description\s*: Imports xPaaS templates"
      And stdout should match "Enabled\s*: false"
      And stdout should match "Priority\s*: 0"
      And stdout should match "Url\s*:"
      And stdout should match "Openshift Version\s*:"

  @quick
  Scenario: Uninstalling an add-on
     When executing "minishift addons uninstall anyuid" succeeds
     Then executing "minishift addons list" succeeds
      And stdout should not contain "anyuid"

  @quick
  Scenario: Installing add-on from a folder
  Note: working directory when executing Minishift commands is: out/test-run.
     When file from "https://raw.githubusercontent.com/minishift/minishift/master/addons/anyuid/anyuid.addon" is downloaded into location "download/anyuid"
      And executing "minishift addons install download/anyuid" succeeds
     Then executing "minishift addons list" succeeds
      And stdout should contain "anyuid"

  @quick
  Scenario: Applying add-on when Minishift is not running
    Given Minishift has state "Does Not Exist"
     When executing "minishift addons apply anyuid" succeeds
     Then stdout should contain
      """
      Running this command requires an existing 'minishift' VM, but no VM is defined.
      """

  @quick
  Scenario: Removing add-on when Minishift is not running
  After issue no. 1377 is resolved, please change the expected stdout to:
  Minishift should return: "Running this command requires an existing 'minishift' VM, but no VM is defined."
    Given Minishift has state "Does Not Exist"
     When executing "minishift addons remove anyuid" succeeds
     Then stdout should contain
      """
      Unable to remove addon 'anyuid'. No anyuid.addon.remove file is found.
      """

  @quick
  Scenario: Installing default add-ons manually
     When executing "minishift addons install --defaults" succeeds
     Then stdout should contain
      """
      Default add-ons 'anyuid, admin-user, xpaas, registry-route, che, eap-cd, htpasswd-identity-provider, admissions-webhook' installed
      """
     When executing "minishift addons list" succeeds
     Then stdout should contain "admin-user"
     Then stdout should contain "anyuid"
     Then stdout should contain "registry-route"
     Then stdout should contain "xpaas"
     Then stdout should contain "che"
     Then stdout should contain "htpasswd-identity-provider"
     Then stdout should contain "eap-cd"
     Then stdout should contain "admissions-webhook"

  @minishift-only @quick
  Scenario: Default add-ons are not enabled by default during installation
     When executing "minishift addons list"
     Then stdout should match "admin-user\s*: disabled\s*P\(0\)"
      And stdout should match "anyuid\s*: disabled\s*P\(0\)"
      And stdout should match "registry-route\s*: disabled\s*P\(0\)"
      And stdout should match "xpaas\s*: disabled\s*P\(0\)"
      And stdout should match "che\s*: disabled\s*P\(0\)"
      And stdout should match "htpasswd-identity-provider\s*: disabled\s*P\(0\)"
      And stdout should match "admissions-webhook\s*: disabled\s*P\(0\)"

  @quick
  Scenario: Enabling not installed add-on
     When executing "minishift addons enable absent-addon"
     Then stdout should contain
      """
      No add-on with the name 'absent-addon' is installed.
      """

  @quick
  Scenario: Enabling installed add-on
     When executing "minishift addons enable xpaas" succeeds
     Then stdout should contain "Add-on 'xpaas' enabled"
      And executing "minishift addons list" succeeds
      And stdout should match "xpaas\s*: enabled\s*P\(0\)"

  @quick
  Scenario: Enabling installed add-on with priority
     When executing "minishift addons enable anyuid --priority 5" succeeds
     Then stdout should contain "Add-on 'anyuid' enabled"
      And executing "minishift addons list" succeeds
      And stdout should match "anyuid\s*: enabled\s*P\(5\)"

  Scenario: Starting Minishift with anyuid and xpaas add-ons enabled
  The addons are being applied in correct order.
    Given Minishift has state "Does Not Exist"
      And image caching is disabled
     When executing "minishift start" succeeds
     Then Minishift should have state "Running"
     Then stdout should match
      """
      Applying addon 'xpaas'[\S\s]*Applying addon 'anyuid'
      """

  @minishift-only
  Scenario: Disabled add-ons were not applied during the startup
    Given Minishift has state "Running"
      And stdout should not contain "Applying addon 'registry-route'"
      And stdout should not contain "Applying addon 'admin-user'"

  Scenario: Disabling enabled add-on
     When executing "minishift addons disable xpaas" succeeds
     Then stdout should contain
      """
      Add-on 'xpaas' disabled
      """
     When executing "minishift addons list" succeeds
     Then stdout should match "xpaas\s*: disabled"

  Scenario: Disabling disabled add-on
     When executing "minishift addons disable xpaas" succeeds
     Then stdout should contain
      """
      Add-on 'xpaas' is already disabled
      """
     When executing "minishift addons list" succeeds
     Then stdout should match "xpaas\s*: disabled"

  Scenario: Disabling not installed add-on
     When executing "minishift addons disable absent-addon"
     Then stdout should contain
      """
      No add-on with the name 'absent-addon' is installed.
      """

  Scenario: Applying enabled add-on which was not applied during the startup
    Given Minishift has state "Running"
     When executing "minishift addons enable registry-route" succeeds
     Then executing "minishift addons apply registry-route" succeeds
      And stdout should contain
      """
      Add-on 'registry-route' created docker-registry route.
      """

  @minishift-only
  Scenario: Applying disabled add-on which was not applied during the startup
    Given Minishift has state "Running"
     Then executing "minishift addons apply admin-user" succeeds
      And stdout should contain
      """
      Applying addon 'admin-user
      """

  Scenario: Removing applied add-on
    Given Minishift has state "Running"
     When executing "minishift addons remove admin-user" succeeds
     Then stdout should contain "Removing addon 'admin-user'"
      And stdout should contain "admin user deleted"

  Scenario: Stopping Minishift
    Given Minishift has state "Running"
     When executing "minishift stop" succeeds
     Then Minishift should have state "Stopped"

  Scenario: Deleting minishift
    Given Minishift has state "Stopped"
     When executing "minishift delete --force" succeeds
     Then Minishift should have state "Does Not Exist"
