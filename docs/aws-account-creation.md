# Manual to create access to AWS.

## 1. Get your user id.
* Under your username go to `My Account`.
    * If you are a federated user go to `Security Credentials`.
    * Go to users and find yourself.
    * The ARN contains your id `arn:aws:iam::YOUR_USER_ID:user/YOUR_NAME`
* Remember your ID

## 2. Create a policy.

* Create a new custom policy
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "iam:*",
            "Resource": "arn:aws:iam::INSERT_AWS_USER_ID:user/kubermatic/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "sts:GetFederationToken",
                "ec2:*"
            ],
            "Resource": "*"
        }
    ]
}
```
* Put this into the policy and give a name you like. For this example we'll use `kubermatic-policy`

## 3. Create a new account.

* Check `Generate an access token for each user.`
* Create the user
* Save the `Access Key ID` and the `Secret Access Key` we will need them later.
* Under `Permissions` Attach the policy `kubermatic-policy` on it.

## 4. Connect Kubermatic with the AWS account.
* Click the `+` symbol above the nodes field.
* A setup box will appear. Choose your preferences
* Now type in the `Access Key ID` and the `Secret Access Key` and you are ready to go.
