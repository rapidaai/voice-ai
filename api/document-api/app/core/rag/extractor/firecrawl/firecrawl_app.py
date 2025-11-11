import json
import time

import datetime
import json
from app.core.helper import encrypter
from app.core.rag.extractor.firecrawl.firecrawl_app import FirecrawlApp
from extensions.ext_storage import storage
import requests
from extensions.ext_storage import storage


class FirecrawlApp:
    def __init__(self, api_key=None, base_url=None):
        self.api_key = api_key
        self.base_url = base_url or 'https://api.firecrawl.dev'
        if self.api_key is None and self.base_url == 'https://api.firecrawl.dev':
            raise ValueError('No API key provided')

    def scrape_url(self, url, params=None) -> dict:
        headers = {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {self.api_key}'
        }
        json_data = {'url': url}
        if params:
            json_data.update(params)
        response = requests.post(
            f'{self.base_url}/v0/scrape',
            headers=headers,
            json=json_data
        )
        if response.status_code == 200:
            response = response.json()
            if response['success'] == True:
                data = response['data']
                return {
                    'title': data.get('metadata').get('title'),
                    'description': data.get('metadata').get('description'),
                    'source_url': data.get('metadata').get('sourceURL'),
                    'markdown': data.get('markdown')
                }
            else:
                raise Exception(f'Failed to scrape URL. Error: {response["error"]}')

        elif response.status_code in [402, 409, 500]:
            error_message = response.json().get('error', 'Unknown error occurred')
            raise Exception(f'Failed to scrape URL. Status code: {response.status_code}. Error: {error_message}')
        else:
            raise Exception(f'Failed to scrape URL. Status code: {response.status_code}')

    def crawl_url(self, url, params=None) -> str:
        start_time = time.time()
        headers = self._prepare_headers()
        json_data = {'url': url}
        if params:
            json_data.update(params)
        response = self._post_request(f'{self.base_url}/v0/crawl', json_data, headers)
        if response.status_code == 200:
            job_id = response.json().get('jobId')
            return job_id
        else:
            self._handle_error(response, 'start crawl job')

    def check_crawl_status(self, job_id) -> dict:
        headers = self._prepare_headers()
        response = self._get_request(f'{self.base_url}/v0/crawl/status/{job_id}', headers)
        if response.status_code == 200:
            crawl_status_response = response.json()
            if crawl_status_response.get('status') == 'completed':
                total = crawl_status_response.get('total', 0)
                if total == 0:
                    raise Exception('Failed to check crawl status. Error: No page found')
                data = crawl_status_response.get('data', [])
                url_data_list = []
                for item in data:
                    if isinstance(item, dict) and 'metadata' in item and 'markdown' in item:
                        url_data = {
                            'title': item.get('metadata').get('title'),
                            'description': item.get('metadata').get('description'),
                            'source_url': item.get('metadata').get('sourceURL'),
                            'markdown': item.get('markdown')
                        }
                        url_data_list.append(url_data)
                if url_data_list:
                    file_key = 'website_files/' + job_id + '.txt'
                    if storage.exists(file_key):
                        storage.delete(file_key)
                    storage.save(file_key, json.dumps(url_data_list).encode('utf-8'))
                return {
                    'status': 'completed',
                    'total': crawl_status_response.get('total'),
                    'current': crawl_status_response.get('current'),
                    'data': url_data_list
                }
            else:
                return {
                    'status': crawl_status_response.get('status'),
                    'total': crawl_status_response.get('total'),
                    'current': crawl_status_response.get('current'),
                    'data': []
                }
        else:
            self._handle_error(response, 'check crawl status')

    def _prepare_headers(self):
        return {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {self.api_key}'
        }

    def _post_request(self, url, data, headers, retries=3, backoff_factor=0.5):
        for attempt in range(retries):
            response = requests.post(url, headers=headers, json=data)
            if response.status_code == 502:
                time.sleep(backoff_factor * (2 ** attempt))
            else:
                return response
        return response

    def _get_request(self, url, headers, retries=3, backoff_factor=0.5):
        for attempt in range(retries):
            response = requests.get(url, headers=headers)
            if response.status_code == 502:
                time.sleep(backoff_factor * (2 ** attempt))
            else:
                return response
        return response

    def _handle_error(self, response, action):
        error_message = response.json().get('error', 'Unknown error occurred')
        raise Exception(f'Failed to {action}. Status code: {response.status_code}. Error: {error_message}')






class WebsiteService:

    @classmethod
    def document_create_args_validate(cls, args: dict):
        if 'url' not in args or not args['url']:
            raise ValueError('url is required')
        if 'options' not in args or not args['options']:
            raise ValueError('options is required')
        if 'limit' not in args['options'] or not args['options']['limit']:
            raise ValueError('limit is required')

    @classmethod
    def crawl_url(cls, args: dict) -> dict:
        provider = args.get('provider')
        url = args.get('url')
        options = args.get('options')
        # credentials = ApiKeyAuthService.get_auth_credentials(current_user.current_tenant_id,
        #                                                      'website',
                                                            #  provider)
        if provider == 'firecrawl':
            # decrypt api_key
            # api_key = encrypter.decrypt_token(
            #     tenant_id=current_user.current_tenant_id,
            #     token=credentials.get('config').get('api_key')
            # )
            firecrawl_app = FirecrawlApp(api_key=api_key,
                                         base_url=credentials.get('config').get('base_url', None))
            crawl_sub_pages = options.get('crawl_sub_pages', False)
            only_main_content = options.get('only_main_content', False)
            if not crawl_sub_pages:
                params = {
                    'crawlerOptions': {
                        "includes": [],
                        "excludes": [],
                        "generateImgAltText": True,
                        "limit": 1,
                        'returnOnlyUrls': False,
                        'pageOptions': {
                            'onlyMainContent': only_main_content,
                            "includeHtml": False
                        }
                    }
                }
            else:
                includes = options.get('includes').split(',') if options.get('includes') else []
                excludes = options.get('excludes').split(',') if options.get('excludes') else []
                params = {
                    'crawlerOptions': {
                        "includes": includes if includes else [],
                        "excludes": excludes if excludes else [],
                        "generateImgAltText": True,
                        "limit": options.get('limit', 1),
                        'returnOnlyUrls': False,
                        'pageOptions': {
                            'onlyMainContent': only_main_content,
                            "includeHtml": False
                        }
                    }
                }
                if options.get('max_depth'):
                    params['crawlerOptions']['maxDepth'] = options.get('max_depth')
            job_id = firecrawl_app.crawl_url(url, params)
            # website_crawl_time_cache_key = f'website_crawl_{job_id}'
            time = str(datetime.datetime.now().timestamp())
            # redis_client.setex(website_crawl_time_cache_key, 3600, time)
            return {
                'status': 'active',
                'job_id': job_id
            }
        else:
            raise ValueError('Invalid provider')

    @classmethod
    def get_crawl_status(cls, job_id: str, provider: str) -> dict:
        # credentials = ApiKeyAuthService.get_auth_credentials(current_user.current_tenant_id,
        #                                                      'website',
        #                                                      provider)
        if provider == 'firecrawl':
            # decrypt api_key
            api_key = encrypter.decrypt_token(
                tenant_id=current_user.current_tenant_id,
                token=credentials.get('config').get('api_key')
            )
            firecrawl_app = FirecrawlApp(api_key=api_key,
                                         base_url=credentials.get('config').get('base_url', None))
            result = firecrawl_app.check_crawl_status(job_id)
            crawl_status_data = {
                'status': result.get('status', 'active'),
                'job_id': job_id,
                'total': result.get('total', 0),
                'current': result.get('current', 0),
                'data': result.get('data', [])
            }
            if crawl_status_data['status'] == 'completed':
                website_crawl_time_cache_key = f'website_crawl_{job_id}'
                # start_time = redis_client.get(website_crawl_time_cache_key)
                # if start_time:
                #     end_time = datetime.datetime.now().timestamp()
                #     time_consuming = abs(end_time - float(start_time))
                #     crawl_status_data['time_consuming'] = f"{time_consuming:.2f}"
                #     redis_client.delete(website_crawl_time_cache_key)
        else:
            raise ValueError('Invalid provider')
        return crawl_status_data

    @classmethod
    def get_crawl_url_data(cls, job_id: str, provider: str, url: str, tenant_id: str) -> dict | None:
        # credentials = ApiKeyAuthService.get_auth_credentials(tenant_id,
        #                                                      'website',
        #                                                      provider)
        if provider == 'firecrawl':
            file_key = 'website_files/' + job_id + '.txt'
            if storage.exists(file_key):
                data = storage.load_once(file_key)
                if data:
                    data = json.loads(data.decode('utf-8'))
            else:
                # decrypt api_key
                # api_key = encrypter.decrypt_token(
                #     tenant_id=tenant_id,
                #     token=credentials.get('config').get('api_key')
                # )
                firecrawl_app = FirecrawlApp(api_key=api_key,
                                             base_url=credentials.get('config').get('base_url', None))
                result = firecrawl_app.check_crawl_status(job_id)
                if result.get('status') != 'completed':
                    raise ValueError('Crawl job is not completed')
                data = result.get('data')
            if data:
                for item in data:
                    if item.get('source_url') == url:
                        return item
            return None
        else:
            raise ValueError('Invalid provider')

    @classmethod
    def get_scrape_url_data(cls, provider: str, url: str, tenant_id: str, only_main_content: bool) -> dict | None:
        # credentials = ApiKeyAuthService.get_auth_credentials(tenant_id,
        #                                                      'website',
        #                                                      provider)
        if provider == 'firecrawl':
            # decrypt api_key
            # api_key = encrypter.decrypt_token(
            #     tenant_id=tenant_id,
            #     token=credentials.get('config').get('api_key')
            # )
            firecrawl_app = FirecrawlApp(api_key=api_key,
                                         base_url=credentials.get('config').get('base_url', None))
            params = {
                'pageOptions': {
                    'onlyMainContent': only_main_content,
                    "includeHtml": False
                }
            }
            result = firecrawl_app.scrape_url(url, params)
            return result
        else:
            raise ValueError('Invalid provider')
