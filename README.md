# ChatGPT Slack アプリ
このSlackアプリは、OpenAIのChatGPTモデルを統合して、Slackチャンネルで自然言語会話を可能にするものです。
## 必要条件
- 管理者アクセス権限を持つSlackワークスペース
- Go 1.20以上のバージョン
- 有効なOpenAI APIキー
- IAMおよびS3アクセスが可能なAWSアカウント
## S3 バケットの作成
SAMアーティファクトを保存するために、S3バケットを作成する必要があります。以下のコマンドを実行して作成できます。
$ aws s3 mb s3://&lt;bucket-name&gt;
&lt;bucket-name&gt;は、バケットの一意の名前に置き換えてください。
## デプロイ
1. このリポジトリをクローンします。
$ git clone https://github.com/omeroid/chat-gpt
$ cd chatgpt
2. OpenAI APIキーを取得するには、[ここ](&lt;<https://beta.openai.com/docs/api-reference/authentication&gt;>)の手順に従ってください。
3. APIキーを環境変数としてエクスポートします。
$ export OPENAI_API=&lt;your_api_key_here&gt;
4. AWS SAM CLIを[ここ](&lt;<https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html&gt;>)の手順に従ってインストールします。
5. SAMアプリケーションをビルドしてパッケージ化します。
$ sam build
$ sam package --output-template-file packaged.yaml --s3-bucket &lt;bucket-name&gt;
&lt;bucket-name&gt;は、ステップ1で作成したS3バケットの名前に置き換えてください。
6. SAMアプリケーションをデプロイします。
$ sam deploy --template-file packaged.yaml --stack-name &lt;stack-name&gt; --capabilities CAPABILITY_IAM --parameter-overrides 'SlackToken=$(SLACK_TOKEN) OpenAIAPI=$(OPENAI_API)'
&lt;stack-name&gt;は、SAMスタックの一意の名前に置き換え、SLACK_TOKENにはSlackアプリのトークン、OPENAI_APIにはOpenAI APIキーを設定してください。
7. Slackワークスペースで新しいアプリを作成し、[ここ](&lt;<https://api.slack.com/start/overview#creating&gt;>)の手順に従ってインストールします。
8. ボットとchat:writeスコープをSlackアプリに追加します。
9. 設定を構成するには、デプロイされたSAMエンドポイントのURLをリクエストURLに設定します。
10. アプリをテストするには、アプリがインストールされたチャンネルにメッセージを送信します。
## 使用方法
1. Slack上でChatGPTを利用したいチャンネルに入ります。
2. 下記のコマンドを入力してください。
   
   @ChatGPT &lt;your_message&gt;
   
   <sup>※1</sup>
   ※1：@ChatGPT は、ChatGPTがBotとして導入される際のBot nameまたはUser name に合わせて、適宜変更する必要があります。
3. ChatGPTが応答します。ChatGPTは、入力されたメッセージに対する応答を自動的に生成するAI Chatbotです。
4. ChatGPTは生成されたメッセージをスレッドに投稿します。スレッド内部では、過去の会話を踏まえた上で回答が行われます。
5. スレッド内部で使用される情報は、ChatGPTに対してメンションが行われているメッセージと、ChatGPTからの回答のみです。
6. ChatGPTを使用するには、最初にシステムプロンプトを指定することも可能です。最初のメッセージは、以下のようにしてシステムプロンプトを指定して送信します。
   
   @ChatGPT あなたは***です 
       system_prompt: &lt;system_prompt_message&gt;
   
   <sup>※2</sup>
   ※2：&lt;system_prompt_message&gt; は、生成された応答のシステムプロンプトとして使用されるテキストを指定します。
7. 会話の開始には、`[image]` というプレフィックスをつけて最初のメッセージを送信する必要があります。以下は、例です。
   
   @ChatGPT [image] PCを開いている人
   
   <sup>※3</sup>
   ※3：`[image]` をプレフィックスとして使用することで、ChatGPTが送信したメッセージに対して画像を生成します。
8. ChatGPTが返信した内容を無視する場合は、メッセージの先頭に [ignore] というプレフィックスをつけることができます。以下は、例です。
   
   @ChatGPT [ignore] I don't need the answer, thanks.
   
以上が、追加の仕様を加えたSlackでChatGPTの利用方法になります。必要に応じて、上記の内容を変更してください。
