package endpoints

// THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT.

type endpointStruct struct {
	Version   int
	Endpoints map[string]endpointEntry
}

type endpointEntry struct {
	Endpoint      string
	SigningRegion string
}

var endpointsMap = endpointStruct{
	Version: 2,
	Endpoints: map[string]endpointEntry{
		"*/*": {
			Endpoint: "{service}.{region}.amazonaws.com",
		},
		"*/cloudfront": {
			Endpoint:      "cloudfront.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"*/cloudsearchdomain": {
			Endpoint:      "",
			SigningRegion: "us-east-1",
		},
		"*/data.iot": {
			Endpoint:      "",
			SigningRegion: "us-east-1",
		},
		"*/ec2metadata": {
			Endpoint:      "http://169.254.169.254/latest",
			SigningRegion: "us-east-1",
		},
		"*/iam": {
			Endpoint:      "iam.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"*/importexport": {
			Endpoint:      "importexport.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"*/route53": {
			Endpoint:      "route53.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"*/sts": {
			Endpoint:      "sts.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"*/waf": {
			Endpoint:      "waf.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"ap-northeast-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"ap-northeast-2/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"ap-southeast-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"ap-southeast-2/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"cn-north-1/*": {
			Endpoint: "{service}.{region}.amazonaws.com.cn",
		},
		"eu-central-1/s3": {
			Endpoint: "{service}.{region}.amazonaws.com",
		},
		"eu-west-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"sa-east-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"us-east-1/s3": {
			Endpoint: "s3.amazonaws.com",
		},
		"us-east-1/sdb": {
			Endpoint:      "sdb.amazonaws.com",
			SigningRegion: "us-east-1",
		},
		"us-gov-west-1/iam": {
			Endpoint: "iam.us-gov.amazonaws.com",
		},
		"us-gov-west-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"us-gov-west-1/sts": {
			Endpoint: "sts.us-gov-west-1.amazonaws.com",
		},
		"us-west-1/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
		"us-west-2/s3": {
			Endpoint: "s3-{region}.amazonaws.com",
		},
	},
}
