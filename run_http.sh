#!/bin/bash

threadCount=50
totalRequests=10000

makeRazorxRequest()
{
	response=$(curl -s -w "\n%{http_code}" --location --request GET 'https://razorx.dev.razorpay.in/v1/evaluate?environment=stage&feature_flag=pp_payment_required_amount_quantity_check&id=IY9DXu40I53Ocw&mode=live' --header 'Authorization: Basic cnpwX2FwaTphcGlfcGFzc3dvcmQ=' --connect-timeout 2 --max-time 10)

	http_code=$(tail -n1 <<< "$response")
	content=$(sed '$ d' <<< "$response")

	if [[ "$http_code" != "200" ]]
	then
		echo 'got non 200 status:' $http_code $content
	fi

	# echo $http_code '--' $content
}

export -f makeRazorxRequest

seq 1 $totalRequests | xargs -I % -P $threadCount bash -c 'makeRazorxRequest "{}"'
