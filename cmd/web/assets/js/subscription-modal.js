window.subscriptionModal = function(plansData) {
    return {
        cryptos: [
            { Symbol: 'BTC', Name: 'Bitcoin', Rate: 68000.0 },
            { Symbol: 'ETH', Name: 'Ethereum', Rate: 3400.0 },
            { Symbol: 'USDT', Name: 'Tether', Rate: 1.0 }
        ],
        step: 'plans',
        plans: plansData || [],
        selectedPlan: null,
        selectedCrypto: null,
        estimatedCryptoAmount: 0,
        isEstimating: false,
        paymentDetails: null,

        selectPlan(plan) {
            this.selectedPlan = plan;
            this.step = 'crypto';
            // Default to BTC
            this.selectedCrypto = this.cryptos.find(c => c.Symbol === 'BTC');
            this.updateEstimate();
        },

        selectCrypto(cryptoSymbol) {
            this.selectedCrypto = this.cryptos.find(c => c.Symbol === cryptoSymbol);
            this.updateEstimate();
        },

        updateEstimate() {
            if (!this.selectedPlan || !this.selectedCrypto) return;
            this.isEstimating = true;

            // Logic: use Discount if available, otherwise Price
            const priceValue = this.selectedPlan.Discount > 0 ?
                this.selectedPlan.Discount :
                this.selectedPlan.Price;

            // Simulate API delay for realism
            setTimeout(() => {
                this.estimatedCryptoAmount = (priceValue / this.selectedCrypto.Rate).toFixed(8);
                this.isEstimating = false;
            }, 400);
        },

        proceedToPayment() {
            this.paymentDetails = {
                Address: 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh',
                QRCodeURL: `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=bitcoin:bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh?amount=${this.estimatedCryptoAmount}`
            };
            this.step = 'payment';
        },

        copyToClipboard(text) {
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(text);
            } else {
                // Fallback for non-https/older browsers
                let textArea = document.createElement("textarea");
                textArea.value = text;
                document.body.appendChild(textArea);
                textArea.select();
                document.execCommand('copy');
                document.body.removeChild(textArea);
            }
        }
    }
}