FROM debian:stretch-slim

RUN apt-get update \
 && apt-get install -y wget pwgen \
 && apt-get clean all

COPY * /opt/

#All game data is stored under /data. Assets, executables, config, logs
VOLUME ["/data"]

#OpenArena needs this one port. 
EXPOSE 27960/udp 

#This is environments you can give to Docker.
#OA_STARTMAP sets the first map to load. This is required because the server does not start until a map is loaded.
ENV OA_STARTMAP oasago2
ENV OA_PORT 27960
ENV OA_ROTATE_LOGS 1

RUN chmod +x /opt/*.sh

#This is the default start path
CMD ["./opt/openarena_start_script.sh"]

#Can be started like this:
#docker run -it -e "OA_STARTMAP=dm4ish" -e "OA_PORT=27960" --rm -p 27960:27960/udp -v openarena_data:/data sago007/openarena

#Be warned that the port number must be changed in all three places for the server to appear in the serverlist (2016-05-02). I have not examinated if this is a bug or design flaw in ioquake3 or Docker but the server port is not reported correctly to the master server. 

#To change the config you can start a bash shell, install vim (or other editor) and edit the config:
#Start with: docker run -it --rm -v openarena_data:/data --user 0 sago007/openarena bash
#And then execute:
#apt-get install -y vim
#vim /data/openarena/baseoa/server_config_sample.cfg
