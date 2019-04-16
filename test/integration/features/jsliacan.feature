@quick
Feature: Jsliacan	
	As a user I can do stupid things and get stupid answers.

	Scenario: User can check version of Minishift
		When executing "minishift version" succeeds
		Then stdout should contain "minishift v1."

	Scenario: User can start minishift
		Given Minishift has state "Does Not Exist"
		When executing "minishift start" succeeds
		Then Minishift should have state "Running"
		And stderr should be empty

	Scenario: User can ask meaningless things
		Given Minishift has state "Running"
		When executing "minishift feedme"
		Then stderr should contain "Error: unknown"

	Scenario: Create a Python app
		Given Minishift has state "Running"
		When executing "oc new-app python~https://github.com/GrahamDumpleton/os-sample-python.git --name mypythonapp" succeeds
		And executing "oc expose svc/mypythonapp" succeeds

	Scenario: Delete the Python app
		When executing "oc delete all --selector app=mypythonapp" succeeds
		Then stdout should contain "deleted"

	Scenario: User goes away
		Given Minishift has state "Running"
		When executing "minishift stop" succeeds
		Then Minishift should have state "Stopped"
		When executing "minishift delete --force --clear-cache" succeeds
		Then Minishift should have state "Does Not Exist"
		