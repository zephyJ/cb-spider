RESTSERVER=localhost

#정상 동작
#Name으로 조회
curl -X GET http://$RESTSERVER:1024/keypair/mcb-keypair?connection_name=aws-config01 |json_pp