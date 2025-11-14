pkill -f "./bin/node" && pkill -f "signaling" && killall -9 node signaling 2>/dev/null; echo "Processos encerrados"
sleep 2 && lsof -ti:8080,8081,8082,9000,9001,9002,9003 2>/dev/null | xargs kill -9 2>/dev/null; echo "Portas liberadas"
