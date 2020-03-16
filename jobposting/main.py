import json
import os
import random

from flask import Request
from slack import WebClient
from slack.web.classes.blocks import DividerBlock, SectionBlock, ContextBlock
from slack.web.classes.dialogs import DialogBuilder
from slack.web.classes.messages import Message
from slack.web.classes.objects import MarkdownTextObject, Option

SLACK_VERIFICATION_TOKEN = os.environ['SLACK_API_KEY']

POSTING_CHANNEL = "slack-apps"

CALLBACK_ID = "JOB-POST-CALLBACK-ID"

POSITION_PARAMETER = "position"
COMPANY_PARAMETER = "company"
LOCATION_PARAMETER = "location"
MIN_SALARY_PARAMETER = "min_salary"
MAX_SALARY_PARAMETER = "max_salary"
OFFER_LINK_PARAMETER = "offer_link"
EXTRA_DATA_TEXT_PARAMETER = "extra_data_text"
TALK_WITH_PARAMETER = "talk_with"


LOCATION_OPTIONS = ['Barcelona', 'Andorra', 'Remote']
SKIN_TONES_RANGE = (2, 7)

# Slack client for Web API requests
slack_client = WebClient(token=SLACK_VERIFICATION_TOKEN, ssl=True)

def post_job_offer(position: str,
                   company: str,
                   location: str,
                   min_salary: int,
                   max_salary: int,
                   link: str,
                   posting_user: str,
                   extra_data: str = None):
    message_blocks = []

    message_blocks.append(DividerBlock())

    message_blocks.append(SectionBlock(text=MarkdownTextObject(
        text=" :male-technologist::skin-tone-{male_skin_tone}: | :female-technologist::skin-tone-{female_skin_tone}: {position} @ *{company}* :round_pushpin: _{location}_ :moneybag: {min_salary}K-{max_salary}K :moneybag:".format(
            position=position,
            company=company,
            location=location,
            min_salary=min_salary,
            max_salary=max_salary,
            male_skin_tone=random.randrange(*SKIN_TONES_RANGE),
            female_skin_tone=random.randrange(*SKIN_TONES_RANGE))
    )))

    if extra_data:
        message_blocks.append(DividerBlock())
        message_blocks.append(SectionBlock(text=MarkdownTextObject(
            text=extra_data
        )))
        message_blocks.append(DividerBlock())

    message_blocks.append(SectionBlock(text="<{}/>".format(link)))

    message_blocks.append(
        ContextBlock(elements=[MarkdownTextObject(text="_Talk with @{} for more info._".format(posting_user))]))

    work_order_message = Message(text="New Job offer!", blocks=message_blocks)
    slack_client.chat_postMessage(channel=POSTING_CHANNEL, **work_order_message.to_dict())


def post_modal(trigger_id: str, user_id: str):
    # TODO: Use modals when builder available. (Multiselect is available for example)
    dialog_builder = DialogBuilder()
    dialog_builder.title("Post Job Offer")
    dialog_builder.text_field(name=POSITION_PARAMETER, label="Position", hint="the open position. e.g: Software Engineer")
    dialog_builder.text_field(name=COMPANY_PARAMETER, label="Company", hint="the company with the open position.")
    dialog_builder.static_selector(name=LOCATION_PARAMETER, label="Location", options=[Option.from_single_value(o) for o in LOCATION_OPTIONS])
    dialog_builder.text_field(name=MIN_SALARY_PARAMETER, label="Min Salary [k€]", placeholder="20", hint="minimum yearly salary, in thousands of euros.")
    dialog_builder.text_field(name=MAX_SALARY_PARAMETER, label="Max Salary [k€]", placeholder="40", hint="maximum yearly salary, in thousands of euros.")
    dialog_builder.text_area(name=EXTRA_DATA_TEXT_PARAMETER,label="Extra data", placeholder="perks, equity, etc...", hint="extra useful information, markdown allowed.", optional=True)
    dialog_builder.text_field(name=OFFER_LINK_PARAMETER, label="Offer URL", hint="external job offer URL.")
    dialog_builder.user_selector(name=TALK_WITH_PARAMETER, label="Talk with: ", value=user_id)
    dialog_builder.submit_label("Post")
    dialog_builder.callback_id(CALLBACK_ID)
    slack_client.dialog_open(trigger_id=trigger_id, view=dialog_builder.to_dict(), dialog=dialog_builder.to_dict())

def handle_request(request: Request):
    form_data = request.form
    # post dialog to user
    if 'trigger_id' in form_data:
        trigger_id = form_data['trigger_id']
        posting_user = form_data['user_id']
        post_modal(trigger_id=trigger_id, user_id=posting_user)
    elif 'payload' in form_data:
        payload = json.loads(form_data['payload'])
        type = payload['type']
        callback_id = payload['callback_id']
        if type == 'dialog_submission' and callback_id == CALLBACK_ID:
            submission = payload['submission']
            posting_user_id = submission[TALK_WITH_PARAMETER]
            user_info = slack_client.users_info(user=posting_user_id)
            posting_user = user_info.data['user']['name']

            position = submission[POSITION_PARAMETER]
            company = submission[COMPANY_PARAMETER]
            location = submission[LOCATION_PARAMETER]
            min_salary = submission[MIN_SALARY_PARAMETER]
            max_salary = submission[MAX_SALARY_PARAMETER]

            extra_data = submission.get(EXTRA_DATA_TEXT_PARAMETER)
            link = submission[OFFER_LINK_PARAMETER]
            post_job_offer(position=position, company=company, location=location, min_salary=min_salary,
                           max_salary=max_salary, link=link, posting_user=posting_user, extra_data=extra_data)
    return ""  # response is required to be empty

