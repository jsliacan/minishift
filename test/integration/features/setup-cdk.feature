@setup-cdk
Feature: Setup CDK
  As a user I can able to setup CDK

  # --force option is needed since we are doing setup-cdk before running each feature file (testsuite.go)
  Scenario: User can able to setup CDK with default location
    When executing "minishift setup-cdk --force" succeeds
    Then stdout should match "Setting up CDK 3 on host using '.*' as Minishift's home directory"
    And stdout should match "Copying minishift-rhel7.iso to '.*(\/|\\)cache(\/|\\)iso(\/|\\)minishift-rhel7.iso'"
    And stdout should match "Copying oc(.exe)? to '.*(\/|\\)cache(\/|\\)oc(\/|\\)v[0-9]+\.[0-9]+\.[0-9]+(.?[0-9]+?\.?[0-9]+)?(\/|\\)(linux|windows|darwin)(\/|\\)oc(.exe)?"
    And stdout should match "Creating configuration file '.*(\/|\\)config(\/|\\)config.json'"
    And stdout should match "Creating marker file '.*(\/|\\)cdk'"
    And stdout should contain
     """
     Default add-ons anyuid, admin-user, xpaas, registry-route, che, htpasswd-identity-provider, eap-cd installed
     Default add-ons anyuid, admin-user, xpaas enabled
     CDK 3 setup complete.
     """

  Scenario: User can able to setup CDK with non-default location
    When executing "minishift setup-cdk --force --minishift-home test" succeeds
    Then stdout should match "Setting up CDK 3 on host using '.*test' as Minishift's home directory"
    And stdout should match "Copying minishift-rhel7.iso to '.*test(\/|\\)cache(\/|\\)iso(\/|\\)minishift-rhel7.iso'"
    And stdout should match "Copying oc(.exe)? to '.*test(\/|\\)cache(\/|\\)oc(\/|\\)v[0-9]+\.[0-9]+\.[0-9]+(.?[0-9]+?\.?[0-9]+)?(\/|\\)(linux|windows|darwin)(\/|\\)oc(.exe)?"
    And stdout should match "Creating configuration file '.*test(\/|\\)config(\/|\\)config.json'"
    And stdout should match "Creating marker file '.*test(\/|\\)cdk'"
    And stdout should contain
     """
     Default add-ons anyuid, admin-user, xpaas, registry-route, che, htpasswd-identity-provider, eap-cd installed
     Default add-ons anyuid, admin-user, xpaas enabled
     CDK 3 setup complete.
     """
    And deleting directory "test" succeeds

  Scenario: User can able to set different VM driver
    When executing "minishift setup-cdk --force --default-vm-driver foo" succeeds
    Then JSON config file ".minishift/config/config.json" contains key "vm-driver" with value matching "foo"

  Scenario: User can setup CDK for different profile
    When executing "minishift setup-cdk" succeeds
    And executing "minishift setup-cdk --profile foo" succeeds
    Then stdout should match "Setting up CDK 3 on host using '.*(\/|\\)profiles(\/|\\)foo' as Minishift's home directory"
    And stdout should match "Creating configuration file '.*(\/|\\)profiles(\/|\\)foo(\/|\\)config(\/|\\)config.json'"
    And stdout should match "Creating marker file '.*(\/|\\)profiles(\/|\\)foo(\/|\\)cdk'"
    And stdout should contain
     """
     Default add-ons anyuid, admin-user, xpaas, registry-route, che, htpasswd-identity-provider, eap-cd installed
     Default add-ons anyuid, admin-user, xpaas enabled
     CDK 3 setup complete.
     """
