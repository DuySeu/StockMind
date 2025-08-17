import logging
import random

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


def execute_OTP(tool_name, tool_input):
    # Execute OTP tool
    if tool_name is None:
        return None
    
    # Generate OTP code
    otp_digits = [str(random.randint(0, 9)) for _ in range(6)]
    
    # Format as "**XXX-XXX**"
    otp_code = f"**{(''.join(otp_digits[:3]))}-{(''.join(otp_digits[3:]))}**"

    # logger.info(f"Preprocessing for pie data {chartData}")
    print("Executing OTP tool...")
    return {**tool_input, "otp": otp_code}
