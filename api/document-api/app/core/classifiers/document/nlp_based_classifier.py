"""
Copyright (c) 2024 Prashant Srivastav <prashant@rapida.ai>
All rights reserved.

This code is licensed under the MIT License. You may obtain a copy of the License at
https://opensource.org/licenses/MIT.

Unless required by applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions
and limitations under the License.

"""

from collections import Counter
from typing import List

from app import nlp
from app.core.classifiers.abstract_classifier import AbstractClassifier, Domain

from app.core.rag.models.document import Document

fintech_words = [
    "banking" "account",
    "payment",
    "credit",
    "bond",
    "deposit",
    "ratio",
    "loan",
    "bank",
    "rate",
    "card",
    "currency",
    "debt",
    "market",
    "interest",
    "statement",
    "report",
    "risk",
    "order",
    "analysis",
    "audit",
    "balance",
    "transaction",
    "transfer",
    "foreign",
    "compliance",
    "capital",
    "reserve",
    "management",
    "merchant",
    "cash",
    "trading",
    "financial",
    "debit",
    "ach",
    "liquidity",
    "discount",
    "check",
    "fee",
    "digital",
    "data",
    "reconciliation",
    "clearing",
    "lending",
    "yield",
    "bill",
    "fund",
    "equity",
    "tier",
    "accounting",
    "premium",
    "mobile",
    "branch",
    "direct",
    "wire",
    "swift",
    "code",
    "exchange",
    "remittance",
    "escrow",
    "underwriting",
    "regulation",
    "basel",
    "policy",
    "quantitative",
    "money",
    "chargeback",
    "gateway",
    "wallet",
    "settlement",
    "nostro",
    "interbank",
    "funding",
    "acquisition",
    "security",
    "tax",
    "payments",
    "withdrawal",
    "overdraft",
    "mortgage",
    "atm",
    "pin",
    "savings",
    "teller",
    "iban",
    "collateral",
    "lien",
    "appraisal",
    "amortization",
    "principal",
    "compound",
    "simple",
    "bureau",
    "bankruptcy",
    "foreclosure",
    "repossession",
    "refinancing",
    "consolidation",
    "collection",
    "fraud",
    "identity",
    "theft",
    "phishing",
    "skimming",
    "kyc",
    "laundering",
    "insured",
    "uninsured",
    "asset",
    "wealth",
    "investment",
    "custodian",
    "federal",
    "monetary",
    "banknote",
    "coin",
    "cashier",
    "prepaid",
    "charge",
    "secured",
    "advance",
    "grace",
    "period",
    "pos",
    "cryptocurrency",
    "blockchain",
    "branchless",
    "neobank",
    "fintech",
    "ebitda",
    "capitalization",
    "dividend",
    "stock",
    "split",
    "rights",
    "bonus",
    "share",
    "buyback",
    "derivatives",
    "futures",
    "options",
    "swaps",
    "forwards",
    "hedging",
    "syndicated",
    "consortium",
    "bridge",
    "financing",
    "securitization",
    "asset-backed",
    "mortgage-backed",
    "collateralized",
    "obligation",
    "swap",
    "repurchase",
    "forex",
    "margin",
    "arbitrage",
    "carry",
    "trade",
]


class NLPBasedClassifier(AbstractClassifier):
    def classify(self, docs: List[Document]) -> Domain:
        text = "\n".join(st.page_content for st in docs)
        doc = nlp.en_core_web_sm(text)

        # Filter words based on specific criteria
        filtered_words = [
            (
                token.lemma_.lower()
                if token.tag_ in {"NNS", "NNPS"}
                else token.text.lower()
            )
            for token in doc
            if token.text.strip()
            and not token.is_stop
            and not token.is_punct
            and not token.is_digit
            and len(token.text) > 1
        ]

        # Get the most used words
        word_freq = Counter(filtered_words)
        common_words = [word for word, freq in word_freq.most_common(100)]

        # Check how many words found in financial dictionary
        non_financial_words = set(common_words) - set(fintech_words)

        if len(non_financial_words) <= 10:
            return Domain.FINANCIAL

        return Domain.UNKNOWN
