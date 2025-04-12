document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM fully loaded - initializing application');
    
    // Debug helper
    window.debugApp = function() {
        console.log('Debugging application state:');
        console.log('- File upload element:', document.querySelector('.file-upload'));
        console.log('- File input element:', document.querySelector('.file-upload-input'));
        console.log('- Calculate form:', document.getElementById('calculate-form'));
        console.log('- Result container:', document.getElementById('result-container'));
        
        // Test template download links
        console.log('- CSV template link:', document.querySelector('a[href="/static/templates/lease_template.csv"]'));
        console.log('- Excel template link:', document.querySelector('a[href="/static/templates/lease_template.xlsx"]'));
        
        // Test modal functionality
        console.log('- Help modal:', document.getElementById('help-modal'));
    };
    
    // Debug on load
    setTimeout(function() {
        console.log('Running initial debug check:');
        if (window.debugApp) window.debugApp();
    }, 1000);
    
    // Handle collapsible sections
    const collapsibleHeaders = document.querySelectorAll('.collapse-header');
    
    collapsibleHeaders.forEach(header => {
        header.addEventListener('click', function() {
            const collapseBody = this.nextElementSibling;
            
            if (collapseBody.classList.contains('open')) {
                collapseBody.classList.remove('open');
                this.querySelector('.collapse-icon').textContent = '+';
            } else {
                collapseBody.classList.add('open');
                this.querySelector('.collapse-icon').textContent = '-';
            }
        });
    });
    
    // Handle file upload UI enhancement
    const fileUpload = document.querySelector('.file-upload');
    const fileInput = document.querySelector('.file-upload-input');
    const fileText = document.querySelector('.file-upload-text');
    
    if (fileUpload && fileInput) {
        console.log('File upload elements found, attaching listeners');
        
        // Click to upload
        fileUpload.addEventListener('click', function(e) {
            if (e.target !== fileInput) { // Prevent double trigger if clicking directly on input
                fileInput.click();
            }
        });
        
        // Drag and drop functionality
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            fileUpload.addEventListener(eventName, preventDefaults, false);
        });
        
        function preventDefaults(e) {
            e.preventDefault();
            e.stopPropagation();
        }
        
        ['dragenter', 'dragover'].forEach(eventName => {
            fileUpload.addEventListener(eventName, highlight, false);
        });
        
        ['dragleave', 'drop'].forEach(eventName => {
            fileUpload.addEventListener(eventName, unhighlight, false);
        });
        
        function highlight() {
            fileUpload.classList.add('highlight');
            fileUpload.style.borderColor = 'var(--accent-color)';
        }
        
        function unhighlight() {
            fileUpload.classList.remove('highlight');
            fileUpload.style.borderColor = 'var(--border-light)';
        }
        
        fileUpload.addEventListener('drop', handleDrop, false);
        
        function handleDrop(e) {
            const dt = e.dataTransfer;
            const files = dt.files;
            
            if (files.length > 0) {
                fileInput.files = files;
                fileText.textContent = `Selected: ${files[0].name}`;
                fileUpload.style.borderColor = 'var(--accent-color)';
            }
        }
        
        // Handle file selection via input
        fileInput.addEventListener('change', function() {
            console.log('File selected via input:', this.files);
            if (this.files.length > 0) {
                fileText.textContent = `Selected: ${this.files[0].name}`;
                fileUpload.style.borderColor = 'var(--accent-color)';
            } else {
                fileText.textContent = 'Click or drag a file here to upload';
                fileUpload.style.borderColor = 'var(--border-light)';
            }
        });
    } else {
        console.warn('File upload elements not found in the DOM');
        console.log('fileUpload:', fileUpload);
        console.log('fileInput:', fileInput);
    }
    
    // Handle AJAX form submission with fetch API
    const calculateForm = document.getElementById('calculate-form');
    const resultContainer = document.getElementById('result-container');
    
    if (calculateForm) {
        console.log('Calculate form found, attaching submit listener');
        
        calculateForm.addEventListener('submit', async function(e) {
            // Only prevent default for AJAX, allow regular form submission as fallback
            if (window.fetch) {
                e.preventDefault();
            } else {
                return true; // Let the form submit normally if fetch isn't available
            }
            
            console.log('Form submitted via AJAX');
            
            const formData = new FormData(this);
            console.log('Form data:', [...formData.entries()]);
            
            try {
                // Show loading state
                resultContainer.innerHTML = '<div class="card"><p>Processing...</p></div>';
                
                console.log('Sending request to /calculate');
                const response = await fetch('/calculate', {
                    method: 'POST',
                    body: formData
                });
                
                console.log('Response received:', response.status, response.statusText);
                
                if (!response.ok) {
                    throw new Error('Server returned error status: ' + response.status);
                }
                
                const results = await response.json();
                console.log('Results received:', results);
                displayResults(results);
            } catch (error) {
                console.error('Error during calculation:', error);
                resultContainer.innerHTML = `
                    <div class="alert alert-error">
                        <p>Error: ${error.message}</p>
                    </div>
                `;
            }
        });
    } else {
        console.warn('Calculate form not found in the DOM');
    }
    
    // Function to display calculation results
    function displayResults(results) {
        if (!resultContainer) return;
        
        if (results.length === 0) {
            resultContainer.innerHTML = `
                <div class="alert alert-error">
                    <p>No results returned from the calculation.</p>
                </div>
            `;
            return;
        }
        
        let html = `
            <div class="card">
                <div class="card-header">
                    <h2 class="card-title">Calculation Results</h2>
                    <button id="export-btn" class="btn btn-primary">Export to Excel</button>
                </div>
                <p>${results.length} lease(s) processed</p>
            </div>
        `;
        
        // Add individual result cards
        results.forEach((result, index) => {
            html += `
                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">Lease ID: ${result.leaseId || 'Unknown'}</h3>
                    </div>
                    
                    ${result.error ? `
                        <div class="alert alert-error">
                            <p>Error: ${result.error}</p>
                        </div>
                    ` : `
                        <div class="result-summary">
                            <div class="result-row">
                                <span class="result-label">Initial Liability:</span>
                                <span class="result-value">${formatCurrency(result.initialLiability)}</span>
                            </div>
                            <div class="result-row">
                                <span class="result-label">Initial RoU Asset:</span>
                                <span class="result-value">${formatCurrency(result.initialRoUAsset)}</span>
                            </div>
                            <div class="result-row">
                                <span class="result-label">Total Periods:</span>
                                <span class="result-value">${result.liabilitySchedule.length}</span>
                            </div>
                        </div>
                        
                        <div class="collapse-header">
                            <span>Liability Schedule</span>
                            <span class="collapse-icon">+</span>
                        </div>
                        <div class="collapse-body">
                            <table class="table">
                                <thead>
                                    <tr>
                                        <th>Period</th>
                                        <th>Date</th>
                                        <th>Opening Balance</th>
                                        <th>Payment</th>
                                        <th>Interest</th>
                                        <th>Principal</th>
                                        <th>Closing Balance</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${result.liabilitySchedule.map(entry => `
                                        <tr>
                                            <td>${entry.period}</td>
                                            <td>${formatDate(entry.date)}</td>
                                            <td>${formatCurrency(entry.openingBalance)}</td>
                                            <td>${formatCurrency(entry.payment)}</td>
                                            <td>${formatCurrency(entry.interestExpense)}</td>
                                            <td>${formatCurrency(entry.principalRepayment)}</td>
                                            <td>${formatCurrency(entry.closingBalance)}</td>
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>
                        
                        <div class="collapse-header">
                            <span>RoU Asset Schedule</span>
                            <span class="collapse-icon">+</span>
                        </div>
                        <div class="collapse-body">
                            <table class="table">
                                <thead>
                                    <tr>
                                        <th>Period</th>
                                        <th>Date</th>
                                        <th>Opening Balance</th>
                                        <th>Depreciation</th>
                                        <th>Closing Balance</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${result.rouAssetSchedule.map(entry => `
                                        <tr>
                                            <td>${entry.period}</td>
                                            <td>${formatDate(entry.date)}</td>
                                            <td>${formatCurrency(entry.openingBalance)}</td>
                                            <td>${formatCurrency(entry.depreciation)}</td>
                                            <td>${formatCurrency(entry.closingBalance)}</td>
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>
                    `}
                </div>
            `;
        });
        
        resultContainer.innerHTML = html;
        
        // Add event listener to export button
        const exportBtn = document.getElementById('export-btn');
        if (exportBtn) {
            exportBtn.addEventListener('click', function() {
                exportToExcel(results);
            });
        }
        
        // Re-attach event listeners to collapsible elements
        document.querySelectorAll('.collapse-header').forEach(header => {
            header.addEventListener('click', function() {
                const collapseBody = this.nextElementSibling;
                
                if (collapseBody.classList.contains('open')) {
                    collapseBody.classList.remove('open');
                    this.querySelector('.collapse-icon').textContent = '+';
                } else {
                    collapseBody.classList.add('open');
                    this.querySelector('.collapse-icon').textContent = '-';
                }
            });
        });
    }
    
    // Function to export results to Excel
    async function exportToExcel(results) {
        try {
            const response = await fetch('/export', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(results)
            });
            
            if (!response.ok) {
                throw new Error('Error exporting to Excel');
            }
            
            // Create blob from response
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            
            // Create temporary link and trigger download
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = 'ifrs16_calculation_results.xlsx';
            document.body.appendChild(a);
            a.click();
            
            // Clean up
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        } catch (error) {
            alert('Error exporting to Excel: ' + error.message);
        }
    }
    
    // Helper functions
    function formatCurrency(amount) {
        return new Intl.NumberFormat('en-US', {
            style: 'decimal',
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }).format(amount);
    }
    
    function formatDate(dateStr) {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        return date.toISOString().split('T')[0];
    }
}); 