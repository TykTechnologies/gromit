CFN      := aws cloudformation

master: pro.yaml master.json
	$(CFN) validate-template --template-body file://$<
	$(CFN) update-stack --stack-name $@ --parameters file://$@.json --template-body file://$<

del:
	$(CFN) delete-stack --stack-name master
	$(CFN) wait stack-delete-complete --stack-name master

create: pro.yaml master.json
	$(CFN) validate-template --template-body file://$<
	$(CFN) create-stack --stack-name master --parameters file://master.json --template-body file://$<
