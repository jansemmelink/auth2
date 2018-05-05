#!/bin/bash
#
debug=0
function debug()
{
	if [ ${debug} -ne 0 ]
	then
		echo -e $(date "+%Y-%m-%d %H:%M:%S") "DEBUG  " $* >&2
	fi
}

function verbose()
{
	if [ ${debug} -ne 0 ]
	then
		echo -e $(date "+%Y-%m-%d %H:%M:%S") "VERBOSE" $* >&2
	else
		echo -e $(date "+%Y-%m-%d %H:%M:%S") $* >&2
	fi
}

function error()
{
	if [ ${debug} -ne 0 ]
	then
		echo -e $(date "+%Y-%m-%d %H:%M:%S") "ERROR  " $* >&2
	else
		echo -e "ERROR: " $* >&2
	fi
	exit 1
}

host="localhost"
port="3005"
email="test@conor.co.za"
password="Kop4raem"
logout_delay=2
function usage() {
	if [ $# -gt 0 ]; then echo -e "ERROR: $*" >&2; fi
	echo -e "---------------------------------------------------------------------" >&2
	echo -e "Usage: $(basename $0) [options]" >&2
	echo -e "Options:" >&2
	echo -e "\t-d                  Debug mode" >&2
	echo -e "\t-h <host>           Host     (default: ${host})" >&2
	echo -e "\t-p <port>           Port     (default: ${port})" >&2
	echo -e "\t-e <email>          Email    (default: ${email})" >&2
	echo -e "\t-s <password>       Password (default: ${password})" >&2
	echo -e "\t-n <nr seconds>     Logout delay in seconds (default ${logout_delay})" >&2
	echo -e "---------------------------------------------------------------------" >&2
	exit 1
}

while [ $# -gt 0 ]
do
	case $1 in
	"-d") debug=1;;
	"-p") port=$2; shift;;
	"-h") host=$2; shift;;
	"-e") email=$2; shift;;
	"-s") password=$2; shift;;
	"-n") logout_delay=$2; shift;;
	"--help") usage;;
	*) usage "Invalid options: $*"
	esac
	shift
done

[ -z "${host}" ] && error "-h <host> required."
[ -z "${port}" ] && error "-p <port> required."
[ -z "${email}" ] && error "-e <email> required."
[ -z "${password}" ] && error "-s <password> required."

addr="http://${host}:${port}"
debug "addr=${addr}"

function api()
{
	method=$1
	shift
	url=$1
	shift
	file=$1
	shift
	verbose ${method} ${url} file=${file}
	out=$(mktemp)

	# curl options:
	# -s for silent, without progress meter
	# -w to write http code to ...
	# -o to redirect output to file
	if [ -z "${file}" ]
	then
		http_code=$(curl -s -w "%{http_code}" -X${method} ${url} -o ${out})
	else
		http_code=$(curl -s -w "%{http_code}" -X${method} ${url} -d @${file} -o ${out})
	fi

	if [ $? -ne 0 ]
	then
		echo ${http_code},$(cat ${out})
		rm -f ${out}
		error "Failed to ${method} ${url}"
	fi
	
	if [ ${debug} -ne 0 ]
	then
		debug "http_code=${http_code}"
		debug "OUTPUT: $(cat ${out})"
	fi

	if [ ${http_code} -ne 200 ]
	then
		echo ${http_code},$(cat ${out})
		rm -f ${out}
		error "HTTP ${method} ${url} -> ${http_code}"
	fi

	# success
	echo ${http_code},$(cat ${out})
	rm -f ${out}
}

#---------------------------------------------------
# we use email as user name for a person that can login
# if register fail with status 409=HTTP.StatusConflict
# we know the user already exists
#---------------------------------------------------

#---------------------------------------------------
# register the user using email as the user's name
# output is id and temp password
#---------------------------------------------------
verbose "Registering ${email} ..."
t=$(mktemp)
echo "{\"name\":\"${email}\"}" > ${t}
debug "user data in file ${t}: $(cat ${t})"

register_response=$(api POST "${addr}/auth/register" ${t})
http_code=${register_response%%,*}
register_response=${register_response#*,}
debug "http_code=${http_code} response=${register_response}"
if [ ${http_code} -eq 200 ]
then
	#-------------------------------------
	# registration succeeded with code 200
	#-------------------------------------
	id=$(echo ${register_response} | jq '.id' | sed "s/\"//g")
	TempPassword=$(echo ${register_response} | jq '.TempPassword' | sed "s/\"//g")

	verbose "Registered user.id=${id} TempPassword=${TempPassword}"

	#--------------------------------------------------
	# activate with email and TempPassword added to activation url
	#--------------------------------------------------
	debug "Activating with password=${password}"
	activate_res=$(api GET "${addr}/auth/activate?name=${email}&tpw=${TempPassword}&password=${password}")
	http_code=${activate_res%%,*}
	activate_res=${activate_res#*,}
	debug "http_code=${http_code} response=${activate_res}"

	[ ${http_code} -ne 200 ] && error "Failed to activate: ${activate_res}"
	sid=$(echo ${activate_res} | jq '.id' | sed "s/\"//g")

	verbose "Activated and logged in user: session.id==${sid}"
else
	#----------------------------------------------------
	# registration failed
	#----------------------------------------------------
	if [ ${http_code} -eq 409 ]
	then
		verbose "User ${email} already exists: ${user_data}"
		#--------------------------------------------------
		# login with specified password
		#--------------------------------------------------
		debug "Logging in with password=${password}"
		t=$(mktemp)
		echo "{\"name\":\"${email}\",\"password\":\"${password}\"}" > ${t}
		login_res=$(api POST "${addr}/auth/login" ${t})
		http_code=${login_res%%,*}
		login_res=${login_res#*,}
		debug "http_code=${http_code} response=${login_res}"
		
		[ ${http_code} -ne 200 ] && error "Failed to login: ${login_res}"
		sid=$(echo ${login_res} | jq '.id' | sed "s/\"//g")

		verbose "Logged in user: session.id==${sid}"
	else
		error "Registration failed: HTTP ${http_code}: ${register_response}"
	fi
fi

#--------------------------------------------------
# logout from this session
# sleep some seconds, just to show diff in session
# between start and end time
# logout writes no output
#--------------------------------------------------
while [ ${logout_delay} -gt 0 ]
do
	verbose "Waiting ${logout_delay} seconds before logout..."
	sleep 1
	let logout_delay=logout_delay-1
done

api GET "${addr}/auth/logout?id=${sid}"
[ $? -ne 0 ] && error "Failed to logout from session.id=${sid}"
verbose "Logged out of session.id=${sid}"

if [ ! -z "${t}" ]; then rm -f ${t}; unset t; fi
verbose "Done"
