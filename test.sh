docker network create test-net
docker run --name mysql --network=test-net -e MYSQL_ROOT_PASSWORD=test -d mysql:8.0.17
docker exec mysql mysql -uroot -ptest -e "$(cat setup.sql)"
docker run --name --network=test-net pingnstor jrcichra/pingnstor:$TRAVIS_COMMIT
sleep 120
docker exec mysql mysql -uroot -ptest -e "select count(*) from pingnstor.pings where domain = 'google.com'"
docker kill pingnstor