if [ $(mysql -uroot -sse "select count(*) from pingnstor.pings where domain = 'google.com'") -gt 0 ]
then
return 0
else
return 1